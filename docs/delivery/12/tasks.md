# Tasks for PBI 12: Just-in-Time Segment Generation

This document lists all tasks associated with PBI 12.

**Parent PBI**: [PBI 12: Just-in-Time Segment Generation](./prd.md)

## Task Summary

| Task ID | Name | Status | Description |
| :------ | :--- | :----- | :---------- |
| 12-1 | [Add batch configuration options](./12-1.md) | Done | Add BatchSize and TriggerThreshold to StreamingConfig with validation and defaults |
| 12-2 | [Extend StreamSession model for batch tracking](./12-2.md) | Done | Add batch state tracking to StreamSession: batch number, segment range, video position, client positions |
| 12-3 | [Implement position tracking API endpoint](./12-3.md) | Done | Create POST /stream/:channel_id/position endpoint for client position reporting |
| 12-4 | [Modify FFmpeg command builder for batch mode](./12-4.md) | Done | Update BuildHLSCommand to generate fixed segment batches, remove -stream_loop, add segment limiting |
| 12-5 | [Create batch coordinator](./12-5.md) | Done | Implement BatchCoordinator in StreamManager that monitors positions and triggers next batch |
| 12-6 | [Implement batch continuation logic](./12-6.md) | Proposed | Add seamless batch-to-batch continuation with video position calculation and timeline service integration |
| 12-7 | [Update cleanup for batch-aware operation](./12-7.md) | Proposed | Modify segment cleanup to keep N-1 batch during transitions, delete N-2 batch after completion |
| 12-8 | [Remove legacy realtime pacing mode](./12-8.md) | Proposed | Remove RealtimePacing config option and continuous stream monitoring code |
| 12-9 | [Update frontend for position reporting](./12-9.md) | Proposed | Add position reporting to HLS player component that sends current segment every N seconds |
| 12-10 | [Create integration tests for batch system](./12-10.md) | Proposed | Implement integration tests for batch generation, position tracking, triggering, and continuation |



