//! Entry-point for the `dcode-exec` binary.
//!
//! When this CLI is invoked normally, it parses the standard `dcode-exec` CLI
//! options and launches the non-interactive Dcode agent. However, if it is
//! invoked with arg0 as `dcode-linux-sandbox`, we instead treat the invocation
//! as a request to run the logic for the standalone `dcode-linux-sandbox`
//! executable (i.e., parse any -s args and then run a *sandboxed* command under
//! Landlock + seccomp.
//!
//! This allows us to ship a completely separate set of functionality as part
//! of the `dcode-exec` binary.
use clap::Parser;
use dcode_arg0::Arg0DispatchPaths;
use dcode_arg0::arg0_dispatch_or_else;
use dcode_exec::Cli;
use dcode_exec::run_main;
use dcode_utils_cli::CliConfigOverrides;

#[derive(Parser, Debug)]
struct TopCli {
    #[clap(flatten)]
    config_overrides: CliConfigOverrides,

    #[clap(flatten)]
    inner: Cli,
}

fn main() -> anyhow::Result<()> {
    arg0_dispatch_or_else(|arg0_paths: Arg0DispatchPaths| async move {
        let top_cli = TopCli::parse();
        // Merge root-level overrides into inner CLI struct so downstream logic remains unchanged.
        let mut inner = top_cli.inner;
        inner
            .config_overrides
            .raw_overrides
            .splice(0..0, top_cli.config_overrides.raw_overrides);

        run_main(inner, arg0_paths).await?;
        Ok(())
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    use pretty_assertions::assert_eq;

    #[test]
    fn top_cli_parses_resume_prompt_after_config_flag() {
        const PROMPT: &str = "echo resume-with-global-flags-after-subcommand";
        let cli = TopCli::parse_from([
            "dcode-exec",
            "resume",
            "--last",
            "--json",
            "--model",
            "gpt-5.2-dcode",
            "--config",
            "reasoning_level=xhigh",
            "--dangerously-bypass-approvals-and-sandbox",
            "--skip-git-repo-check",
            PROMPT,
        ]);

        let Some(dcode_exec::Command::Resume(args)) = cli.inner.command else {
            panic!("expected resume command");
        };
        let effective_prompt = args.prompt.clone().or_else(|| {
            if args.last {
                args.session_id.clone()
            } else {
                None
            }
        });
        assert_eq!(effective_prompt.as_deref(), Some(PROMPT));
        assert_eq!(cli.config_overrides.raw_overrides.len(), 1);
        assert_eq!(
            cli.config_overrides.raw_overrides[0],
            "reasoning_level=xhigh"
        );
    }
}
