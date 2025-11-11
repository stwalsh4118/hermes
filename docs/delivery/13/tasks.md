# Tasks for PBI 13: Reliable HLS live (4s TS) with Go-managed playlists

This document lists all tasks associated with PBI 13.

**Parent PBI**: [PBI 13: Reliable HLS live (4s TS) with Go-managed playlists](./prd.md)

## Task Summary

| Task ID | Name | Status | Description |
| :------ | :--- | :----- | :---------- |
| 13-1 | [FFmpeg 4s TS stream_segment pipeline](./13-1.md) | Done | Emit 4s MPEG-TS segments with enforced GOP/keyframes (no playlist) |
| 13-2 | [hls-m3u8 package usage guide](./13-2.md) | Done | Produce usage guide and validate API assumptions per policy 2.1.9 |
| 13-3 | [Playlist Manager using hls-m3u8](./13-3.md) | Done | Build sliding-window media playlist with atomic writes |
| 13-4 | [Segment Watcher and pruning](./13-4.md) | Proposed | Detect new segments and prune beyond (window + safety) |
| 13-5 | [Discontinuity detection and tagging](./13-5.md) | Done | Insert #EXT-X-DISCONTINUITY on encoder restarts/timestamp regressions |
| 13-6 | [Observability for HLS pipeline](./13-6.md) | Proposed | Add logs/metrics for cadence, drift, window length, last update time |
| 13-7 | [Update infrastructure API specs](./13-7.md) | Proposed | Document PlaylistManager interfaces in infrastructure API spec |
| 13-8 | [Unit and Integration Tests](./13-8.md) | Proposed | Test playlist window logic and end-to-end pipeline behavior |
| 13-9 | [E2E CoS Test](./13-9.md) | Proposed | Verify PBI acceptance criteria with an end-to-end test plan |


