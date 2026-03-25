//! GitHub Copilot OAuth device code flow.
//!
//! Uses GitHub's standard OAuth device authorization grant to obtain a GitHub
//! access token that can be used as a Bearer token for the Copilot API.

use std::io;
use std::sync::Arc;
use std::time::Duration;
use tokio::sync::Notify;

const CLIENT_ID: &str = "Ov23li8tweQw6odWQebz";
const DEVICE_CODE_URL: &str = "https://github.com/login/device/code";
const ACCESS_TOKEN_URL: &str = "https://github.com/login/oauth/access_token";
/// Extra safety margin added to the server-provided polling interval.
const POLLING_SAFETY_MARGIN: Duration = Duration::from_secs(3);
/// Maximum time to wait for the user to authorize the device.
const MAX_POLL_DURATION: Duration = Duration::from_secs(15 * 60);

/// The data returned by GitHub when starting the device code flow.
#[derive(Debug, Clone)]
pub struct GithubCopilotDeviceCode {
    /// Code shown to the user, to be entered at `verification_uri`.
    pub user_code: String,
    /// URL to open in the browser.
    pub verification_uri: String,
    /// Opaque code used when polling for the access token.
    pub device_code: String,
    /// Minimum seconds between polling attempts.
    pub interval: u64,
}

#[derive(serde::Deserialize)]
struct DeviceCodeResponse {
    device_code: String,
    user_code: String,
    verification_uri: String,
    interval: Option<u64>,
}

#[derive(serde::Deserialize)]
struct TokenResponse {
    access_token: Option<String>,
    error: Option<String>,
    interval: Option<u64>,
}

/// Initiates the GitHub device code flow and returns the codes to show the user.
pub async fn start_github_copilot_auth() -> io::Result<GithubCopilotDeviceCode> {
    let client = reqwest::Client::builder()
        .build()
        .map_err(io::Error::other)?;

    let resp = client
        .post(DEVICE_CODE_URL)
        .header("Accept", "application/json")
        .header("Content-Type", "application/json")
        .body(
            serde_json::json!({
                "client_id": CLIENT_ID,
                "scope": "read:user"
            })
            .to_string(),
        )
        .send()
        .await
        .map_err(io::Error::other)?;

    if !resp.status().is_success() {
        return Err(io::Error::other(format!(
            "GitHub device code request failed with status {}",
            resp.status()
        )));
    }

    let data: DeviceCodeResponse = resp.json().await.map_err(io::Error::other)?;

    Ok(GithubCopilotDeviceCode {
        user_code: data.user_code,
        verification_uri: data.verification_uri,
        device_code: data.device_code,
        interval: data.interval.unwrap_or(5),
    })
}

/// Polls GitHub's token endpoint until the user authorizes or the flow is cancelled.
///
/// Returns the GitHub access token on success.
pub async fn poll_github_copilot_token(
    code: GithubCopilotDeviceCode,
    cancel: Arc<Notify>,
) -> io::Result<String> {
    let client = reqwest::Client::builder()
        .build()
        .map_err(io::Error::other)?;

    let deadline = tokio::time::Instant::now() + MAX_POLL_DURATION;
    let mut interval_secs = code.interval;

    loop {
        let sleep_duration = Duration::from_secs(interval_secs) + POLLING_SAFETY_MARGIN;

        tokio::select! {
            _ = tokio::time::sleep(sleep_duration) => {}
            _ = cancel.notified() => {
                return Err(io::Error::new(io::ErrorKind::Interrupted, "GitHub Copilot login cancelled"));
            }
        }

        if tokio::time::Instant::now() >= deadline {
            return Err(io::Error::other(
                "GitHub Copilot device auth timed out after 15 minutes",
            ));
        }

        let resp = client
            .post(ACCESS_TOKEN_URL)
            .header("Accept", "application/json")
            .header("Content-Type", "application/json")
            .body(
                serde_json::json!({
                    "client_id": CLIENT_ID,
                    "device_code": code.device_code,
                    "grant_type": "urn:ietf:params:oauth:grant-type:device_code"
                })
                .to_string(),
            )
            .send()
            .await
            .map_err(io::Error::other)?;

        if !resp.status().is_success() {
            return Err(io::Error::other(format!(
                "GitHub token poll failed with status {}",
                resp.status()
            )));
        }

        let data: TokenResponse = resp.json().await.map_err(io::Error::other)?;

        if let Some(token) = data.access_token {
            return Ok(token);
        }

        match data.error.as_deref() {
            Some("authorization_pending") => {
                // User hasn't authorized yet; keep polling.
                if let Some(new_interval) = data.interval {
                    interval_secs = new_interval;
                }
            }
            Some("slow_down") => {
                // Server asked us to slow down.
                interval_secs += 5;
            }
            Some(err) => {
                return Err(io::Error::other(format!("GitHub auth error: {err}")));
            }
            None => {
                return Err(io::Error::other("unexpected empty response from GitHub"));
            }
        }
    }
}
