//! SSE processor for the OpenAI-compatible Chat Completions streaming API.

use crate::common::ResponseEvent;
use crate::error::ApiError;
use dcode_client::ByteStream;
use dcode_protocol::models::ContentItem;
use dcode_protocol::models::ResponseItem;
use dcode_protocol::protocol::TokenUsage;
use eventsource_stream::Eventsource;
use futures::StreamExt;
use serde::Deserialize;
use std::collections::HashMap;
use std::time::Duration;
use tokio::sync::mpsc;
use tokio::time::Instant;
use tokio::time::timeout;
use tracing::debug;
use tracing::trace;

#[allow(unused_assignments)]
pub async fn process_chat_sse(
    stream: ByteStream,
    tx_event: mpsc::Sender<Result<ResponseEvent, ApiError>>,
    idle_timeout: Duration,
) {
    let mut stream = stream.eventsource();

    let mut text_buf = String::new();
    let mut tool_call_accumulator: HashMap<usize, ToolCallAccumulator> = HashMap::new();
    let mut response_id = String::new();
    let mut usage: Option<TokenUsage> = None;
    let mut got_done = false;
    let mut finish_reason_seen: Option<String> = None;
    // Set to true after we've emitted OutputItemAdded for the assistant message.
    let mut message_item_started = false;
    // Stable item ID for the assistant message across the whole stream.
    let message_item_id = "msg_chat_0".to_string();

    let _ = tx_event.send(Ok(ResponseEvent::Created)).await;

    loop {
        let start = Instant::now();
        let response = timeout(idle_timeout, stream.next()).await;
        let _ = start.elapsed();

        let sse = match response {
            Ok(Some(Ok(sse))) => sse,
            Ok(Some(Err(e))) => {
                let _ = tx_event
                    .send(Err(ApiError::Stream(e.to_string())))
                    .await;
                return;
            }
            Ok(None) => {
                // Stream closed; finalize if we already saw [DONE]
                if got_done {
                    finalize_stream(
                        &text_buf,
                        &tool_call_accumulator,
                        finish_reason_seen.as_deref(),
                        &response_id,
                        &message_item_id,
                        usage,
                        &tx_event,
                    )
                    .await;
                } else {
                    let _ = tx_event
                        .send(Err(ApiError::Stream(
                            "chat stream closed before [DONE]".into(),
                        )))
                        .await;
                }
                return;
            }
            Err(_) => {
                let _ = tx_event
                    .send(Err(ApiError::Stream(
                        "idle timeout waiting for chat SSE".into(),
                    )))
                    .await;
                return;
            }
        };

        let data = sse.data.trim();
        trace!("Chat SSE: {data}");

        if data == "[DONE]" {
            got_done = true;
            // Finalize immediately on [DONE] without waiting for stream close
            finalize_stream(
                &text_buf,
                &tool_call_accumulator,
                finish_reason_seen.as_deref(),
                &response_id,
                &message_item_id,
                usage,
                &tx_event,
            )
            .await;
            return;
        }

        let chunk: ChatCompletionChunk = match serde_json::from_str(data) {
            Ok(v) => v,
            Err(e) => {
                debug!("Failed to parse chat chunk: {e}, data: {data}");
                continue;
            }
        };

        if response_id.is_empty() {
            response_id = chunk.id.clone();
        }

        if let Some(u) = chunk.usage {
            usage = Some(TokenUsage {
                input_tokens: u.prompt_tokens,
                cached_input_tokens: 0,
                output_tokens: u.completion_tokens,
                reasoning_output_tokens: 0,
                total_tokens: u.total_tokens,
            });
        }

        for choice in chunk.choices {
            let delta = choice.delta;

            if let Some(text) = delta.content {
                if !text.is_empty() {
                    // Emit OutputItemAdded once before the first text delta so
                    // the core loop has an active_item to attach deltas to.
                    if !message_item_started {
                        message_item_started = true;
                        let item = ResponseItem::Message {
                            id: Some(message_item_id.clone()),
                            role: "assistant".into(),
                            content: vec![],
                            end_turn: None,
                            phase: None,
                        };
                        if tx_event
                            .send(Ok(ResponseEvent::OutputItemAdded(item)))
                            .await
                            .is_err()
                        {
                            return;
                        }
                    }
                    text_buf.push_str(&text);
                    if tx_event
                        .send(Ok(ResponseEvent::OutputTextDelta(text)))
                        .await
                        .is_err()
                    {
                        return;
                    }
                }
            }

            if let Some(tool_calls) = delta.tool_calls {
                for tc in tool_calls {
                    let acc = tool_call_accumulator
                        .entry(tc.index)
                        .or_insert_with(ToolCallAccumulator::default);
                    if let Some(id) = tc.id {
                        acc.id = id;
                    }
                    if let Some(func) = tc.function {
                        if let Some(name) = func.name {
                            acc.name = name;
                        }
                        if let Some(args) = func.arguments {
                            acc.arguments.push_str(&args);
                        }
                    }
                }
            }

            if let Some(reason) = choice.finish_reason {
                if reason == "length" {
                    let _ = tx_event
                        .send(Err(ApiError::ContextWindowExceeded))
                        .await;
                    return;
                }
                finish_reason_seen = Some(reason);
            }
        }
    }
}

async fn finalize_stream(
    text_buf: &str,
    tool_call_accumulator: &HashMap<usize, ToolCallAccumulator>,
    finish_reason: Option<&str>,
    response_id: &str,
    message_item_id: &str,
    usage: Option<TokenUsage>,
    tx_event: &mpsc::Sender<Result<ResponseEvent, ApiError>>,
) {
    let _ = finish_reason;

    // Emit accumulated text as a message item (done)
    if !text_buf.is_empty() {
        let item = ResponseItem::Message {
            id: Some(message_item_id.to_string()),
            role: "assistant".into(),
            content: vec![ContentItem::OutputText {
                text: text_buf.to_string(),
            }],
            end_turn: None,
            phase: None,
        };
        if tx_event
            .send(Ok(ResponseEvent::OutputItemDone(item)))
            .await
            .is_err()
        {
            return;
        }
    }

    // Emit accumulated tool calls
    let mut indices: Vec<usize> = tool_call_accumulator.keys().copied().collect();
    indices.sort();
    for idx in indices {
        let acc = &tool_call_accumulator[&idx];
        if acc.name.is_empty() {
            continue;
        }
        let item = ResponseItem::FunctionCall {
            id: None,
            name: acc.name.clone(),
            namespace: None,
            arguments: acc.arguments.clone(),
            call_id: if acc.id.is_empty() {
                format!("call_{idx}")
            } else {
                acc.id.clone()
            },
        };
        if tx_event
            .send(Ok(ResponseEvent::OutputItemDone(item)))
            .await
            .is_err()
        {
            return;
        }
    }

    let _ = tx_event
        .send(Ok(ResponseEvent::Completed {
            response_id: response_id.to_string(),
            token_usage: usage,
        }))
        .await;
}

#[derive(Default)]
struct ToolCallAccumulator {
    id: String,
    name: String,
    arguments: String,
}

#[derive(Debug, Deserialize)]
struct ChatCompletionChunk {
    id: String,
    #[serde(default)]
    choices: Vec<ChunkChoice>,
    #[serde(default)]
    usage: Option<ChunkUsage>,
}

#[derive(Debug, Deserialize)]
struct ChunkChoice {
    delta: ChunkDelta,
    #[serde(default)]
    finish_reason: Option<String>,
}

#[derive(Debug, Deserialize, Default)]
struct ChunkDelta {
    #[serde(default)]
    content: Option<String>,
    #[serde(default)]
    tool_calls: Option<Vec<ToolCallDelta>>,
}

#[derive(Debug, Deserialize)]
struct ToolCallDelta {
    index: usize,
    #[serde(default)]
    id: Option<String>,
    #[serde(default)]
    function: Option<FunctionDelta>,
}

#[derive(Debug, Deserialize)]
struct FunctionDelta {
    #[serde(default)]
    name: Option<String>,
    #[serde(default)]
    arguments: Option<String>,
}

#[derive(Debug, Deserialize)]
struct ChunkUsage {
    #[serde(default)]
    prompt_tokens: i64,
    #[serde(default)]
    completion_tokens: i64,
    #[serde(default)]
    total_tokens: i64,
}
