import { Page, expect } from '@playwright/test';

/**
 * Wait for an element to appear and optionally verify its text content
 */
export async function waitForElement(
  page: Page,
  selector: string,
  options?: { text?: string; timeout?: number }
) {
  const element = page.locator(selector);
  await element.waitFor({ state: 'visible', timeout: options?.timeout || 5000 });

  if (options?.text) {
    await expect(element).toContainText(options.text);
  }

  return element;
}

/**
 * Wait for element to disappear
 */
export async function waitForElementToDisappear(
  page: Page,
  selector: string,
  timeout = 5000
) {
  await page.waitForSelector(selector, { state: 'hidden', timeout });
}

/**
 * Fill form field and verify the value was set
 */
export async function fillAndVerify(
  page: Page,
  selector: string,
  value: string
) {
  await page.fill(selector, value);
  const inputValue = await page.inputValue(selector);
  expect(inputValue).toBe(value);
}

/**
 * Click and wait for navigation or action to complete
 */
export async function clickAndWait(
  page: Page,
  selector: string,
  waitForSelector?: string,
  timeout = 5000
) {
  await page.click(selector);

  if (waitForSelector) {
    await page.waitForSelector(waitForSelector, { state: 'visible', timeout });
  }
}

/**
 * Get text content of all matching elements
 */
export async function getAllTextContents(page: Page, selector: string): Promise<string[]> {
  const elements = page.locator(selector);
  const count = await elements.count();
  const texts: string[] = [];

  for (let i = 0; i < count; i++) {
    const text = await elements.nth(i).textContent();
    if (text) {
      texts.push(text.trim());
    }
  }

  return texts;
}

/**
 * Check if element exists without throwing error
 */
export async function elementExists(page: Page, selector: string): Promise<boolean> {
  const element = page.locator(selector);
  const count = await element.count();
  return count > 0;
}

/**
 * Wait for API response matching a pattern
 */
export async function waitForApiResponse(
  page: Page,
  urlPattern: string | RegExp,
  timeout = 10000
): Promise<any> {
  const response = await page.waitForResponse(
    (response) => {
      const url = response.url();
      if (typeof urlPattern === 'string') {
        return url.includes(urlPattern);
      }
      return urlPattern.test(url);
    },
    { timeout }
  );

  return response.json();
}

/**
 * Mock API endpoint with specific response
 */
export async function mockApiEndpoint(
  page: Page,
  urlPattern: string | RegExp,
  responseData: any,
  statusCode = 200
) {
  await page.route(urlPattern, (route) => {
    route.fulfill({
      status: statusCode,
      contentType: 'application/json',
      body: JSON.stringify(responseData),
    });
  });
}

/**
 * Wait for multiple selectors to be visible
 */
export async function waitForAll(
  page: Page,
  selectors: string[],
  timeout = 5000
) {
  await Promise.all(
    selectors.map((selector) =>
      page.waitForSelector(selector, { state: 'visible', timeout })
    )
  );
}

/**
 * Retry an action until it succeeds or times out
 */
export async function retryUntilSuccess<T>(
  action: () => Promise<T>,
  options?: {
    maxAttempts?: number;
    delayMs?: number;
    onRetry?: (attempt: number) => void;
  }
): Promise<T> {
  const maxAttempts = options?.maxAttempts || 3;
  const delayMs = options?.delayMs || 1000;

  for (let attempt = 1; attempt <= maxAttempts; attempt++) {
    try {
      return await action();
    } catch (error) {
      if (attempt === maxAttempts) {
        throw error;
      }

      if (options?.onRetry) {
        options.onRetry(attempt);
      }

      await new Promise((resolve) => setTimeout(resolve, delayMs));
    }
  }

  throw new Error('Retry failed: maximum attempts exceeded');
}

/**
 * Take screenshot with consistent naming
 */
export async function takeScreenshot(
  page: Page,
  name: string,
  options?: { fullPage?: boolean }
) {
  const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
  const filename = `screenshots/${name}-${timestamp}.png`;

  await page.screenshot({
    path: filename,
    fullPage: options?.fullPage || false,
  });

  return filename;
}

/**
 * Wait for loading state to complete
 */
export async function waitForLoadingToComplete(page: Page, timeout = 10000) {
  // Wait for common loading indicators to disappear
  const loadingSelectors = [
    '.loading',
    '.spinner',
    '[data-loading="true"]',
    '.skeleton',
  ];

  for (const selector of loadingSelectors) {
    const element = page.locator(selector);
    const count = await element.count();

    if (count > 0) {
      await element.first().waitFor({ state: 'hidden', timeout });
    }
  }
}

/**
 * Scroll element into view
 */
export async function scrollIntoView(page: Page, selector: string) {
  await page.locator(selector).scrollIntoViewIfNeeded();
}

/**
 * Get attribute value from element
 */
export async function getAttribute(
  page: Page,
  selector: string,
  attributeName: string
): Promise<string | null> {
  return page.locator(selector).getAttribute(attributeName);
}

/**
 * Check if checkbox/radio is checked
 */
export async function isChecked(page: Page, selector: string): Promise<boolean> {
  return page.locator(selector).isChecked();
}

/**
 * Select option from dropdown
 */
export async function selectOption(
  page: Page,
  selector: string,
  value: string
) {
  await page.selectOption(selector, value);
}

/**
 * Press key combination
 */
export async function pressKeys(page: Page, keys: string) {
  await page.keyboard.press(keys);
}

/**
 * Wait for text to appear anywhere on the page
 */
export async function waitForText(
  page: Page,
  text: string,
  timeout = 5000
) {
  await page.waitForSelector(`text=${text}`, { timeout });
}

/**
 * Clear local storage and session storage
 */
export async function clearStorage(page: Page) {
  await page.evaluate(() => {
    localStorage.clear();
    sessionStorage.clear();
  });
}

/**
 * Get local storage item
 */
export async function getLocalStorageItem(
  page: Page,
  key: string
): Promise<string | null> {
  return page.evaluate((key) => localStorage.getItem(key), key);
}

/**
 * Set local storage item
 */
export async function setLocalStorageItem(
  page: Page,
  key: string,
  value: string
) {
  await page.evaluate(
    ({ key, value }) => localStorage.setItem(key, value),
    { key, value }
  );
}
