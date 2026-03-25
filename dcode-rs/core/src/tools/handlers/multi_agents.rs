//! Implements the collaboration tool surface for spawning and managing sub-agents.
//!
//! This handler translates model tool calls into `AgentControl` operations and keeps spawned
//! agents aligned with the live turn that created them. Sub-agents start from the turn's effective
//! config, inherit runtime-only state such as provider, approval policy, sandbox, and cwd, and
//! then optionally layer role-specific config on top.

use crate::agent::AgentStatus;
use crate::agent::agent_resolver::resolve_agent_target;
use crate::agent::agent_resolver::resolve_agent_targets;
use crate::agent::exceeds_thread_spawn_depth_limit;
use crate::dcode::Session;
use crate::dcode::TurnContext;
use crate::function_tool::FunctionCallError;
use crate::tools::context::ToolInvocation;
use crate::tools::context::ToolOutput;
use crate::tools::context::ToolPayload;
pub(crate) use crate::tools::handlers::multi_agents_common::*;
use crate::tools::handlers::parse_arguments;
use crate::tools::registry::ToolHandler;
use crate::tools::registry::ToolKind;
use async_trait::async_trait;
use dcode_protocol::ThreadId;
use dcode_protocol::models::ResponseInputItem;
use dcode_protocol::openai_models::ReasoningEffort;
use dcode_protocol::protocol::CollabAgentInteractionBeginEvent;
use dcode_protocol::protocol::CollabAgentInteractionEndEvent;
use dcode_protocol::protocol::CollabAgentRef;
use dcode_protocol::protocol::CollabAgentSpawnBeginEvent;
use dcode_protocol::protocol::CollabAgentSpawnEndEvent;
use dcode_protocol::protocol::CollabCloseBeginEvent;
use dcode_protocol::protocol::CollabCloseEndEvent;
use dcode_protocol::protocol::CollabResumeBeginEvent;
use dcode_protocol::protocol::CollabResumeEndEvent;
use dcode_protocol::protocol::CollabWaitingBeginEvent;
use dcode_protocol::protocol::CollabWaitingEndEvent;
use dcode_protocol::user_input::UserInput;
use serde::Deserialize;
use serde::Serialize;
use serde_json::Value as JsonValue;

pub(crate) use close_agent::Handler as CloseAgentHandler;
pub(crate) use resume_agent::Handler as ResumeAgentHandler;
pub(crate) use send_input::Handler as SendInputHandler;
pub(crate) use spawn::Handler as SpawnAgentHandler;
pub(crate) use wait::Handler as WaitAgentHandler;

pub mod close_agent;
mod resume_agent;
mod send_input;
mod spawn;
pub(crate) mod wait;

#[cfg(test)]
#[path = "multi_agents_tests.rs"]
mod tests;
