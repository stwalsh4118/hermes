# Tasks for PBI 14: Custom HLS Playlist Writer/Reader

This document lists all tasks associated with PBI 14.

**Parent PBI**: [PBI 14: Custom HLS Playlist Writer/Reader](./prd.md)

## Task Summary

| Task ID | Name | Status | Description |
| :------ | :--- | :----- | :---------- |
| 14-1 | [Design playlistManager data structure and interface implementation](./14-1.md) | Done | Design the improved Manager interface and playlistManager struct |
| 14-2 | [Implement AddSegment() with sliding window logic and file cleanup](./14-2.md) | Done | Implement segment addition with manual sliding window and file deletion |
| 14-3 | [Implement Write() method with direct m3u8 text generation and atomic writes](./14-3.md) | Done | Generate m3u8 format directly as text with atomic file writes |
| 14-4 | [Implement remaining Manager interface methods](./14-4.md) | Proposed | Implement SetDiscontinuityNext, GetCurrentSegments, HealthCheck, etc. |
| 14-5 | [Add unit tests for all playlist manager methods](./14-5.md) | Proposed | Comprehensive unit tests with >80% coverage |
| 14-6 | [Add integration test with real HLS playback verification](./14-6.md) | Proposed | Test with HLS.js to verify playlist is playable |
| 14-7 | [Update StreamManager to use new playlistManager](./14-7.md) | Proposed | Replace library-based implementation with custom one |
| 14-8 | [Test with real streams and verify no panics or errors](./14-8.md) | Proposed | End-to-end testing with actual streaming |
| 14-9 | [Remove hls-m3u8 library dependency](./14-9.md) | Proposed | Remove library from go.mod and clean up imports |

