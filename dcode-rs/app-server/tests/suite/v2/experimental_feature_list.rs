use std::time::Duration;

use anyhow::Result;
use app_test_support::McpProcess;
use app_test_support::to_response;
use dcode_app_server_protocol::ExperimentalFeature;
use dcode_app_server_protocol::ExperimentalFeatureListParams;
use dcode_app_server_protocol::ExperimentalFeatureListResponse;
use dcode_app_server_protocol::ExperimentalFeatureStage;
use dcode_app_server_protocol::JSONRPCResponse;
use dcode_app_server_protocol::RequestId;
use dcode_core::config::ConfigBuilder;
use dcode_features::FEATURES;
use dcode_features::Stage;
use pretty_assertions::assert_eq;
use tempfile::TempDir;
use tokio::time::timeout;

const DEFAULT_TIMEOUT: Duration = Duration::from_secs(10);

#[tokio::test]
async fn experimental_feature_list_returns_feature_metadata_with_stage() -> Result<()> {
    let dcode_home = TempDir::new()?;
    let config = ConfigBuilder::default()
        .dcode_home(dcode_home.path().to_path_buf())
        .fallback_cwd(Some(dcode_home.path().to_path_buf()))
        .build()
        .await?;
    let mut mcp = McpProcess::new(dcode_home.path()).await?;

    timeout(DEFAULT_TIMEOUT, mcp.initialize()).await??;

    let request_id = mcp
        .send_experimental_feature_list_request(ExperimentalFeatureListParams::default())
        .await?;

    let response: JSONRPCResponse = timeout(
        DEFAULT_TIMEOUT,
        mcp.read_stream_until_response_message(RequestId::Integer(request_id)),
    )
    .await??;

    let actual = to_response::<ExperimentalFeatureListResponse>(response)?;
    let expected_data = FEATURES
        .iter()
        .map(|spec| {
            let (stage, display_name, description, announcement) = match spec.stage {
                Stage::Experimental {
                    name,
                    menu_description,
                    announcement,
                } => (
                    ExperimentalFeatureStage::Beta,
                    Some(name.to_string()),
                    Some(menu_description.to_string()),
                    Some(announcement.to_string()),
                ),
                Stage::UnderDevelopment => {
                    (ExperimentalFeatureStage::UnderDevelopment, None, None, None)
                }
                Stage::Stable => (ExperimentalFeatureStage::Stable, None, None, None),
                Stage::Deprecated => (ExperimentalFeatureStage::Deprecated, None, None, None),
                Stage::Removed => (ExperimentalFeatureStage::Removed, None, None, None),
            };

            ExperimentalFeature {
                name: spec.key.to_string(),
                stage,
                display_name,
                description,
                announcement,
                enabled: config.features.enabled(spec.id),
                default_enabled: spec.default_enabled,
            }
        })
        .collect::<Vec<_>>();
    let expected = ExperimentalFeatureListResponse {
        data: expected_data,
        next_cursor: None,
    };

    assert_eq!(actual, expected);
    Ok(())
}
