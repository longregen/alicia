import { test, expect } from './fixtures';

/**
 * Voice workflow tests using helper functions for cleaner, more maintainable test code.
 * These tests complement the more detailed voice-workflow.spec.ts tests.
 */

test.describe('Voice Workflow (Using Helpers)', () => {
  test.beforeEach(async ({ page, conversationHelpers }) => {
    await page.goto('/');

    // Mock audio APIs
    await page.addInitScript(() => {
      navigator.mediaDevices.getUserMedia = async () => {
        const audioContext = new AudioContext();
        const oscillator = audioContext.createOscillator();
        const destination = audioContext.createMediaStreamDestination();
        oscillator.connect(destination);
        oscillator.start();
        return destination.stream;
      };

      (window as Record<string, unknown>).AudioContext = class MockAudioContext {
        createAnalyser() {
          return {
            fftSize: 256,
            frequencyBinCount: 128,
            connect: () => {},
            getByteFrequencyData: (arr: Uint8Array) => {
              arr.fill(Math.random() * 128);
            },
          };
        }
        createMediaStreamSource() {
          return { connect: () => {} };
        }
        close() {}
      };
    });

    await conversationHelpers.createConversation();
  });

  test('should activate and deactivate voice mode', async ({ voiceHelpers }) => {
    await voiceHelpers.activateVoiceMode();
    expect(await voiceHelpers.isVoiceModeActive()).toBe(true);

    await voiceHelpers.deactivateVoiceMode();
    expect(await voiceHelpers.isVoiceModeActive()).toBe(false);
  });

  test('should control recording', async ({ page, voiceHelpers }) => {
    await voiceHelpers.activateVoiceMode();

    // Start recording
    await voiceHelpers.startRecording();
    const recordBtn = page.locator('.record-btn');
    await expect(recordBtn).toHaveClass(/recording/);

    // Stop recording
    await voiceHelpers.stopRecording();
    await expect(recordBtn).not.toHaveClass(/recording/);
  });

  test('should manage voice selector', async ({ page, voiceHelpers }) => {
    await voiceHelpers.openVoiceSelector();
    await expect(page.locator('.voice-selector-panel')).toBeVisible();

    await voiceHelpers.selectVoice('am_michael');
    const voiceSelect = page.locator('.voice-select');
    expect(await voiceSelect.inputValue()).toBe('am_michael');

    await voiceHelpers.setSpeed(1.5);
    const speedSlider = page.locator('.speed-slider');
    expect(await speedSlider.inputValue()).toBe('1.5');

    await voiceHelpers.closeVoiceSelector();
    await expect(page.locator('.voice-selector-panel')).not.toBeVisible();
  });

  test('should complete full voice interaction with helpers', async ({ page, voiceHelpers }) => {
    // Activate voice mode
    await voiceHelpers.activateVoiceMode();
    expect(await voiceHelpers.isVoiceModeActive()).toBe(true);

    // Configure voice settings
    await voiceHelpers.openVoiceSelector();
    await voiceHelpers.selectVoice('bf_emma');
    await voiceHelpers.setSpeed(1.2);
    await voiceHelpers.closeVoiceSelector();

    // Send a message
    await page.fill('.input-bar input[type="text"]', 'Hello in voice mode');
    await page.click('.input-bar button[type="submit"]');
    await expect(page.locator('div.user').filter({ hasText: 'Hello in voice mode' }).first()).toBeVisible();

    // Deactivate voice mode
    await voiceHelpers.deactivateVoiceMode();
    expect(await voiceHelpers.isVoiceModeActive()).toBe(false);
  });

  test('should handle idempotent operations', async ({ voiceHelpers }) => {
    // Activating when already active should not cause issues
    await voiceHelpers.activateVoiceMode();
    await voiceHelpers.activateVoiceMode();
    expect(await voiceHelpers.isVoiceModeActive()).toBe(true);

    // Deactivating when already inactive should not cause issues
    await voiceHelpers.deactivateVoiceMode();
    await voiceHelpers.deactivateVoiceMode();
    expect(await voiceHelpers.isVoiceModeActive()).toBe(false);
  });

  test('should handle recording state transitions', async ({ page, voiceHelpers }) => {
    await voiceHelpers.activateVoiceMode();

    // Multiple start calls should be idempotent
    await voiceHelpers.startRecording();
    await voiceHelpers.startRecording();

    const recordBtn = page.locator('.record-btn');
    await expect(recordBtn).toHaveClass(/recording/);

    // Multiple stop calls should be idempotent
    await voiceHelpers.stopRecording();
    await voiceHelpers.stopRecording();

    await expect(recordBtn).not.toHaveClass(/recording/);
  });
});
