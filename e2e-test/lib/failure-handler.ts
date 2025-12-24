import { Page, TestInfo } from '@playwright/test';
import { ArtifactCollector } from './artifact-collector';
import { exec } from 'child_process';
import { promisify } from 'util';
import path from 'path';
import fs from 'fs';

const execAsync = promisify(exec);

export class FailureHandler {
  constructor(
    private page: Page,
    private testInfo: TestInfo,
    private collector: ArtifactCollector,
    private artifactDir: string
  ) {}

  async captureAll(): Promise<void> {
    const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
    const testName = this.sanitize(this.testInfo.title);

    await Promise.allSettled([
      this.captureFullPageScreenshot(testName, timestamp),
      this.captureViewportScreenshot(testName, timestamp),
      this.captureDomSnapshot(testName, timestamp),
      this.captureAccessibilityTree(testName, timestamp),
      this.captureLocalStorage(testName, timestamp),
      this.captureNetworkState(testName, timestamp),
      this.triggerVncScreenshot(testName, timestamp),
    ]);

    await this.collector.saveAllLogs();
  }

  private async captureFullPageScreenshot(testName: string, ts: string): Promise<void> {
    const filepath = path.join(
      this.artifactDir,
      'screenshots/browser',
      `failure-${testName}-fullpage-${ts}.png`
    );
    await fs.promises.mkdir(path.dirname(filepath), { recursive: true });
    await this.page.screenshot({ path: filepath, fullPage: true });
    await this.testInfo.attach('failure-fullpage', {
      path: filepath,
      contentType: 'image/png',
    });
  }

  private async captureViewportScreenshot(testName: string, ts: string): Promise<void> {
    const filepath = path.join(
      this.artifactDir,
      'screenshots/browser',
      `failure-${testName}-viewport-${ts}.png`
    );
    await this.page.screenshot({ path: filepath, fullPage: false });
    await this.testInfo.attach('failure-viewport', {
      path: filepath,
      contentType: 'image/png',
    });
  }

  private async captureDomSnapshot(testName: string, ts: string): Promise<void> {
    const filepath = path.join(
      this.artifactDir,
      'dom',
      `failure-${testName}-${ts}.html`
    );
    await fs.promises.mkdir(path.dirname(filepath), { recursive: true });

    const html = await this.page.evaluate(() => {
      const clone = document.documentElement.cloneNode(true) as HTMLElement;

      const importantElements = clone.querySelectorAll('*');
      importantElements.forEach(el => {
        const computed = window.getComputedStyle(el as Element);
        const important = ['display', 'visibility', 'opacity', 'position'];
        const styleStr = important
          .map(prop => `${prop}: ${computed.getPropertyValue(prop)}`)
          .join('; ');
        (el as HTMLElement).setAttribute('data-computed-style', styleStr);
      });

      return clone.outerHTML;
    });

    await fs.promises.writeFile(filepath, html, 'utf-8');
    await this.testInfo.attach('failure-dom', {
      path: filepath,
      contentType: 'text/html',
    });
  }

  private async captureAccessibilityTree(testName: string, ts: string): Promise<void> {
    const filepath = path.join(
      this.artifactDir,
      'dom',
      `failure-${testName}-a11y-${ts}.json`
    );

    try {
      const snapshot = await this.page.accessibility.snapshot();
      await fs.promises.writeFile(
        filepath,
        JSON.stringify(snapshot, null, 2),
        'utf-8'
      );
      await this.testInfo.attach('failure-accessibility', {
        path: filepath,
        contentType: 'application/json',
      });
    } catch {
      // Accessibility tree not always available
    }
  }

  private async captureLocalStorage(testName: string, ts: string): Promise<void> {
    const filepath = path.join(
      this.artifactDir,
      'dom',
      `failure-${testName}-storage-${ts}.json`
    );

    const storage = await this.page.evaluate(() => ({
      localStorage: { ...localStorage },
      sessionStorage: { ...sessionStorage },
    }));

    await fs.promises.writeFile(
      filepath,
      JSON.stringify(storage, null, 2),
      'utf-8'
    );
  }

  private async captureNetworkState(testName: string, ts: string): Promise<void> {
    const filepath = path.join(
      this.artifactDir,
      'dom',
      `failure-${testName}-pending-requests-${ts}.json`
    );

    const pending = await this.page.evaluate(() => {
      const entries = performance.getEntriesByType('resource');
      return entries.map(e => ({
        name: e.name,
        duration: e.duration,
        transferSize: (e as PerformanceResourceTiming).transferSize,
      }));
    });

    await fs.promises.writeFile(
      filepath,
      JSON.stringify(pending, null, 2),
      'utf-8'
    );
  }

  private async triggerVncScreenshot(testName: string, ts: string): Promise<void> {
    if (!process.env.NIXOS_TEST) return;

    try {
      await execAsync(
        `vnc-screenshot.sh failure-${testName}-${ts}`,
        { env: { ...process.env } }
      );
    } catch {
      // VNC screenshot is best-effort
    }
  }

  private sanitize(name: string): string {
    return name.toLowerCase().replace(/[^a-z0-9]/g, '-').substring(0, 50);
  }
}
