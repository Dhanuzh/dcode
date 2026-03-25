//! Implements the MultiAgentV2 collaboration tool surface.

use crate::agent::AgentStatus;
use crate::agent::agent_resolver::resolve_agent_target;
use crate::agent::agent_resolver::resolve_agent_targets;
use crate::agent::exceeds_thread_spawn_depth_limit;
use crate::dcode::Session;
use crate::function_tool::FunctionCallError;
use crate::tools::context::ToolInvocation;
use crate::tools::context::ToolOutput;
use crate::tools::context::ToolPayload;
use crate::tools::handlers::multi_agents_common::*;
use crate::tools::handlers::parse_arguments;
use crate::tools::registry::ToolHandler;
use crate::tools::registry::ToolKind;
use async_trait::async_trait;
use dcode_protocol::AgentPath;
use dcode_protocol::ThreadId;
use dcode_protocol::models::ResponseInputItem;
use dcode_protocol::openai_models::ReasoningEffort;
use dcode_protocol::protocol::CollabAgentInteractionBeginEvent;
use dcode_protocol::protocol::CollabAgentInteractionEndEvent;
use dcode_protocol::protocol::CollabAgentRef;
use dcode_protocol::protocol::CollabAgentSpawnBeginEvent;
use dcode_protocol::protocol::CollabAgentSpawnEndEvent;
use dcode_protocol::protocol::CollabWaitingBeginEvent;
use dcode_protocol::protocol::CollabWaitingEndEvent;
use dcode_protocol::user_input::UserInput;
use serde::Deserialize;
use serde::Serialize;
use serde_json::Value as JsonValue;

pub(crate) use send_input::Handler as SendInputHandler;
pub(crate) use spawn::Handler as SpawnAgentHandler;
pub(crate) use wait::Handler as WaitAgentHandler;

mod send_input;
mod spawn;
pub(crate) mod wait;
