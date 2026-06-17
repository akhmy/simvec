CREATE TABLE IF NOT EXISTS collections (
    name                 String,
    embedder_type        String,
    embedder_full_type   String,
    embedder_credentials String,
    schema               String,
    min_max_rules        String
) ENGINE = ReplacingMergeTree()
ORDER BY name;
