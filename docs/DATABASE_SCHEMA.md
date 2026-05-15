# Database Schema

The primary migration creates all required enterprise CBT tables, shared tenant columns, soft-delete columns, JSON metadata, GIN search indexes, metadata indexes, and module-specific columns.

High-volume operational logs are converted to PostgreSQL range partitioning in `202605130004_partition_high_volume_events.sql`:

- `proctoring_logs`
- `browser_activity_logs`
- `face_detection_logs`
- `recovery_logs`
- `reconnect_logs`

Each partitioned table has a 2026 partition and a default partition, with tenant/time and session indexes. The migration preserves existing rows by renaming the original table to `_legacy`, creating the partitioned parent with the original name, and inserting legacy rows back into the partitioned table.
