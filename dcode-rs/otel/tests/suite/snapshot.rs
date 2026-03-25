use crate::harness::attributes_to_map;
use crate::harness::find_metric;
use dcode_otel::SessionTelemetry;
use dcode_otel::TelemetryAuthMode;
use dcode_otel::metrics::MetricsClient;
use dcode_otel::metrics::MetricsConfig;
use dcode_otel::metrics::Result;
use dcode_protocol::ThreadId;
use dcode_protocol::protocol::SessionSource;
use opentelemetry_sdk::metrics::InMemoryMetricExporter;
use opentelemetry_sdk::metrics::data::AggregatedMetrics;
use opentelemetry_sdk::metrics::data::MetricData;
use pretty_assertions::assert_eq;
use std::collections::BTreeMap;

#[test]
fn snapshot_collects_metrics_without_shutdown() -> Result<()> {
    let exporter = InMemoryMetricExporter::default();
    let config = MetricsConfig::in_memory(
        "test",
        "dcode-cli",
        env!("CARGO_PKG_VERSION"),
        exporter.clone(),
    )
    .with_tag("service", "dcode-cli")?
    .with_runtime_reader();
    let metrics = MetricsClient::new(config)?;

    metrics.counter(
        "dcode.tool.call",
        1,
        &[("tool", "shell"), ("success", "true")],
    )?;

    let snapshot = metrics.snapshot()?;

    let metric = find_metric(&snapshot, "dcode.tool.call").expect("counter metric missing");
    let attrs = match metric.data() {
        AggregatedMetrics::U64(data) => match data {
            MetricData::Sum(sum) => {
                let points: Vec<_> = sum.data_points().collect();
                assert_eq!(points.len(), 1);
                attributes_to_map(points[0].attributes())
            }
            _ => panic!("unexpected counter aggregation"),
        },
        _ => panic!("unexpected counter data type"),
    };

    let expected = BTreeMap::from([
        ("service".to_string(), "dcode-cli".to_string()),
        ("success".to_string(), "true".to_string()),
        ("tool".to_string(), "shell".to_string()),
    ]);
    assert_eq!(attrs, expected);

    let finished = exporter
        .get_finished_metrics()
        .expect("finished metrics should be readable");
    assert!(finished.is_empty(), "expected no periodic exports yet");

    Ok(())
}

#[test]
fn manager_snapshot_metrics_collects_without_shutdown() -> Result<()> {
    let exporter = InMemoryMetricExporter::default();
    let config = MetricsConfig::in_memory("test", "dcode-cli", env!("CARGO_PKG_VERSION"), exporter)
        .with_tag("service", "dcode-cli")?
        .with_runtime_reader();
    let metrics = MetricsClient::new(config)?;
    let manager = SessionTelemetry::new(
        ThreadId::new(),
        "gpt-5.1",
        "gpt-5.1",
        Some("account-id".to_string()),
        None,
        Some(TelemetryAuthMode::ApiKey),
        "test_originator".to_string(),
        true,
        "tty".to_string(),
        SessionSource::Cli,
    )
    .with_metrics(metrics);

    manager.counter(
        "dcode.tool.call",
        1,
        &[("tool", "shell"), ("success", "true")],
    );

    let snapshot = manager.snapshot_metrics()?;
    let metric = find_metric(&snapshot, "dcode.tool.call").expect("counter metric missing");
    let attrs = match metric.data() {
        AggregatedMetrics::U64(data) => match data {
            MetricData::Sum(sum) => {
                let points: Vec<_> = sum.data_points().collect();
                assert_eq!(points.len(), 1);
                attributes_to_map(points[0].attributes())
            }
            _ => panic!("unexpected counter aggregation"),
        },
        _ => panic!("unexpected counter data type"),
    };

    let expected = BTreeMap::from([
        (
            "app.version".to_string(),
            env!("CARGO_PKG_VERSION").to_string(),
        ),
        (
            "auth_mode".to_string(),
            TelemetryAuthMode::ApiKey.to_string(),
        ),
        ("model".to_string(), "gpt-5.1".to_string()),
        ("originator".to_string(), "test_originator".to_string()),
        ("service".to_string(), "dcode-cli".to_string()),
        ("session_source".to_string(), "cli".to_string()),
        ("success".to_string(), "true".to_string()),
        ("tool".to_string(), "shell".to_string()),
    ]);
    assert_eq!(attrs, expected);

    Ok(())
}
