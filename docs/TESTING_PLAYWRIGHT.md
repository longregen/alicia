Key Approaches for Testing LiveKit with Playwright

1. Fake Media Streams (Chromium)

The most common approach uses Chrome launch args:

// playwright.config.ts
launchOptions: {
  args: [
    "--use-fake-device-for-media-stream",
    "--use-fake-ui-for-media-stream",
    "--use-file-for-fake-video-capture=/path/to/video.y4m",  // must be .y4m or .mjpeg
    "--use-file-for-fake-audio-capture=/path/to/audio.wav"
  ]
}

Caveat: Video must be .y4m or .mjpeg format—MP4 won't work.

2. Firefox Alternative

firefoxUserPrefs: {
  "media.navigator.streams.fake": true,
  "media.navigator.permission.disabled": true
}

3. CDP Overrides for Deeper Control

For network condition simulation and ICE candidate monitoring:
const cdpSession = await context.newCDPSession(page);
await cdpSession.send('WebRTC.enable');
// Monitor peer connections, simulate latency, intercept SDP

4. LiveKit's Own Approach

LiveKit docs recommend text-only testing for agents—bypassing actual WebRTC entirely and using mocked tools:
with mock_tools(Agent, {"tool_name": lambda: "mocked_result"}):
    result = await session.run(user_input="test")

For full audio pipeline e2e, they suggest third-party services: Bluejay, Cekura, Coval, or Hamming.

5. Multi-Client Testing

Create two browser contexts with separate CDP sessions to verify bidirectional connections.

---
The honest reality: True end-to-end WebRTC testing remains challenging. Most teams either:
- Use fake media streams to verify the UI/connection setup works
- Mock the WebRTC layer entirely and test business logic separately
- Use specialized WebRTC testing services

Sources:
- https://github.com/microsoft/playwright/issues/2973
- https://github.com/microsoft/playwright/issues/4532
- https://markaicode.com/webrtc-testing-playwright-cdp-overrides/
- https://docs.livekit.io/agents/build/testing/
  
