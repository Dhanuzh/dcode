use crate::protocol::SandboxPolicy;
use crate::spawn::SpawnChildRequest;
use crate::spawn::StdioPolicy;
use crate::spawn::spawn_child_async;
use dcode_network_proxy::NetworkProxy;
use dcode_protocol::permissions::FileSystemSandboxPolicy;
use dcode_protocol::permissions::NetworkSandboxPolicy;
use dcode_sandboxing::landlock::allow_network_for_proxy;
use dcode_sandboxing::landlock::create_linux_sandbox_command_args_for_policies;
use std::collections::HashMap;
use std::path::Path;
use std::path::PathBuf;
use tokio::process::Child;

/// Spawn a shell tool command under the Linux sandbox helper
/// (dcode-linux-sandbox), which defaults to bubblewrap for filesystem
/// isolation plus seccomp for network restrictions.
///
/// Unlike macOS Seatbelt where we directly embed the policy text, the Linux
/// helper is a separate executable. We pass the legacy [`SandboxPolicy`] plus
/// split filesystem/network policies as JSON so the helper can migrate
/// incrementally without breaking older call sites.
#[allow(clippy::too_many_arguments)]
pub async fn spawn_command_under_linux_sandbox<P>(
    dcode_linux_sandbox_exe: P,
    command: Vec<String>,
    command_cwd: PathBuf,
    sandbox_policy: &SandboxPolicy,
    sandbox_policy_cwd: &Path,
    use_legacy_landlock: bool,
    stdio_policy: StdioPolicy,
    network: Option<&NetworkProxy>,
    env: HashMap<String, String>,
) -> std::io::Result<Child>
where
    P: AsRef<Path>,
{
    let file_system_sandbox_policy =
        FileSystemSandboxPolicy::from_legacy_sandbox_policy(sandbox_policy, sandbox_policy_cwd);
    let network_sandbox_policy = NetworkSandboxPolicy::from(sandbox_policy);
    let args = create_linux_sandbox_command_args_for_policies(
        command,
        command_cwd.as_path(),
        sandbox_policy,
        &file_system_sandbox_policy,
        network_sandbox_policy,
        sandbox_policy_cwd,
        use_legacy_landlock,
        allow_network_for_proxy(/*enforce_managed_network*/ false),
    );
    let arg0 = Some("dcode-linux-sandbox");
    spawn_child_async(SpawnChildRequest {
        program: dcode_linux_sandbox_exe.as_ref().to_path_buf(),
        args,
        arg0,
        cwd: command_cwd,
        network_sandbox_policy,
        network,
        stdio_policy,
        env,
    })
    .await
}
