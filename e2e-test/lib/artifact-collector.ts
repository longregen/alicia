import { Page, TestInfo } from '@playwright/test';
import fs from 'fs';
import path from 'path';

export interface ConsoleLogEntry {
  timestamp: string;
  type: 'log' | 'info' | 'warn' | 'error' | 'debug';
  text: string;
  location?: {
    url: string;
    lineNumber: number;
    columnNumber: number;
  };
  args?: string[];
}

export interface NetworkLogEntry {
  timestamp: string;
  type: 'request' | 'response' | 'failure';
  method: string;
  url: string;
  status?: number;
  headers?: Record<string, string>;
  timing?: {
    startTime: number;
    responseTime?: number;
  };
  body?: string;
  error?: string;
}

export class ArtifactCollector {
  private consoleLogs: ConsoleLogEntry[] = [];
  private networkLogs: NetworkLogEntry[] = [];
  private screenshotCounter = 0;
  private artifactDir: string;

  constructor(
    private page: Page,
    private testInfo: TestInfo,
    baseDir?: string
  ) {
    this.artifactDir = baseDir || process.env.ARTIFACT_DIR || './test-results';
    this.setupConsoleCapture();
    this.setupNetworkCapture();
  }

  private setupConsoleCapture(): void {
    this.page.on('console', (msg) => {
      const entry: ConsoleLogEntry = {
        timestamp: new Date().toISOString(),
        type: msg.type() as ConsoleLogEntry['type'],
        text: msg.text(),
        location: msg.location() ? {
          url: msg.location().url,
          lineNumber: msg.location().lineNumber,
          columnNumber: msg.location().columnNumber,
        } : undefined,
      };

      if (msg.args().length > 0) {
        entry.args = msg.args().map(arg => {
          try {
            return arg.toString();
          } catch {
            return '[unserializable]';
          }
        });
      }

      this.consoleLogs.push(entry);
    });

    this.page.on('pageerror', (error) => {
      this.consoleLogs.push({
        timestamp: new Date().toISOString(),
        type: 'error',
        text: `Uncaught exception: ${error.message}`,
        args: [error.stack || ''],
      });
    });
  }

  private setupNetworkCapture(): void {
    this.page.on('request', (request) => {
      this.networkLogs.push({
        timestamp: new Date().toISOString(),
        type: 'request',
        method: request.method(),
        url: request.url(),
        headers: request.headers(),
        timing: { startTime: Date.now() },
      });
    });

    this.page.on('response', async (response) => {
      const entry: NetworkLogEntry = {
        timestamp: new Date().toISOString(),
        type: 'response',
        method: response.request().method(),
        url: response.url(),
        status: response.status(),
        headers: response.headers(),
      };

      if (response.url().includes('/api/') && response.status() !== 304) {
        try {
          const contentType = response.headers()['content-type'] || '';
          if (contentType.includes('application/json')) {
            entry.body = await response.text();
          }
        } catch {
          // Response body not available
        }
      }

      this.networkLogs.push(entry);
    });

    this.page.on('requestfailed', (request) => {
      this.networkLogs.push({
        timestamp: new Date().toISOString(),
        type: 'failure',
        method: request.method(),
        url: request.url(),
        error: request.failure()?.errorText,
      });
    });
  }

  async screenshot(name: string, options?: { fullPage?: boolean }): Promise<string> {
    this.screenshotCounter++;
    const paddedNum = String(this.screenshotCounter).padStart(3, '0');
    const filename = `${paddedNum}-${this.sanitizeFilename(name)}.png`;
    const filepath = path.join(
      this.artifactDir,
      'screenshots/browser',
      filename
    );

    await fs.promises.mkdir(path.dirname(filepath), { recursive: true });
    await this.page.screenshot({
      path: filepath,
      fullPage: options?.fullPage ?? true,
    });

    await this.testInfo.attach(name, {
      path: filepath,
      contentType: 'image/png',
    });

    return filepath;
  }

  async captureDomSnapshot(name: string): Promise<string> {
    const filename = `${this.sanitizeFilename(name)}.html`;
    const filepath = path.join(this.artifactDir, 'dom', filename);

    await fs.promises.mkdir(path.dirname(filepath), { recursive: true });

    const html = await this.page.content();
    await fs.promises.writeFile(filepath, html, 'utf-8');

    await this.testInfo.attach(`${name}-dom`, {
      path: filepath,
      contentType: 'text/html',
    });

    return filepath;
  }

  async saveAllLogs(): Promise<void> {
    const logsDir = path.join(this.artifactDir, 'logs');
    await fs.promises.mkdir(logsDir, { recursive: true });

    const consolePath = path.join(logsDir, 'frontend-console.jsonl');
    await fs.promises.writeFile(
      consolePath,
      this.consoleLogs.map(log => JSON.stringify(log)).join('\n'),
      'utf-8'
    );

    const networkPath = path.join(logsDir, 'network-requests.jsonl');
    await fs.promises.writeFile(
      networkPath,
      this.networkLogs.map(log => JSON.stringify(log)).join('\n'),
      'utf-8'
    );
  }

  async captureFailure(testName: string): Promise<void> {
    const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
    const failureName = `failure-${this.sanitizeFilename(testName)}-${timestamp}`;

    await this.screenshot(failureName, { fullPage: true });
    await this.captureDomSnapshot(failureName);
    await this.saveAllLogs();
  }

  getConsoleErrors(): ConsoleLogEntry[] {
    return this.consoleLogs.filter(log => log.type === 'error');
  }

  getNetworkFailures(): NetworkLogEntry[] {
    return this.networkLogs.filter(log => log.type === 'failure');
  }

  private sanitizeFilename(name: string): string {
    return name
      .toLowerCase()
      .replace(/[^a-z0-9-]/g, '-')
      .replace(/-+/g, '-')
      .replace(/^-|-$/g, '');
  }
}
