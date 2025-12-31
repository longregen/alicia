import { test, expect } from './fixtures';

test.describe('MCP Settings', () => {
  test.beforeEach(async ({ page, mcpHelpers }) => {
    await page.goto('/');
    await mcpHelpers.openSettings();
  });

  test('should navigate to settings', async ({ page }) => {
    // Verify settings panel is visible
    const mcpSettings = page.locator('.mcp-settings');
    await expect(mcpSettings).toBeVisible();

    // Verify header is present
    const header = page.locator('.mcp-settings h2:has-text("MCP Server Settings")');
    await expect(header).toBeVisible();
  });

  test('should display empty state when no servers configured', async ({ page }) => {
    // Assuming fresh state with no servers
    // Scope the selector to MCP settings to avoid matching other empty states
    const emptyState = page.locator('.mcp-settings .empty-state');

    // Check if either empty state is shown or servers list is shown
    const hasServers = await page.locator('.servers-list .server-card').count() > 0;

    if (!hasServers) {
      await expect(emptyState).toBeVisible();
      await expect(emptyState).toContainText('No MCP servers configured');
    }
  });

  test('should add an MCP server', async ({ page, mcpHelpers }) => {
    const serverName = `test-server-${Date.now()}`;
    const command = 'npx';
    const args = '-y, @modelcontextprotocol/server-filesystem, /tmp';

    await mcpHelpers.addServer(serverName, command, args);

    // Verify server appears in the list
    const serverCard = page.locator(`.server-card:has-text("${serverName}")`);
    await expect(serverCard).toBeVisible();

    // Verify server details
    await expect(serverCard.locator('.detail-value:has-text("stdio")')).toBeVisible();
    await expect(serverCard.locator(`.detail-value:has-text("${command}")`)).toBeVisible();
  });

  test('should show server status', async ({ page, mcpHelpers }) => {
    const serverName = `test-server-${Date.now()}`;

    await mcpHelpers.addServer(serverName, 'npx', '-y, @modelcontextprotocol/server-filesystem, /tmp');

    // Wait for server to connect or show status
    const serverCard = page.locator(`.server-card:has-text("${serverName}")`);
    const statusBadge = serverCard.locator('.status-badge');

    await expect(statusBadge).toBeVisible();

    // Status should be one of: Connected, Error, or Disconnected
    const statusText = await statusBadge.textContent();
    expect(['Connected', 'Error', 'Disconnected']).toContain(statusText);
  });

  test('should view tools from a server', async ({ page, mcpHelpers }) => {
    const serverName = `test-server-${Date.now()}`;

    await mcpHelpers.addServer(serverName, 'npx', '-y, @modelcontextprotocol/server-filesystem, /tmp');

    // Wait for server card to appear and tools to load
    const serverCard = page.locator(`.server-card:has-text("${serverName}")`);
    await serverCard.waitFor({ state: 'visible' });

    // Wait for the tools section to appear (it only appears if server.tools.length > 0)
    const toolsSection = serverCard.locator('.tools-section');
    await toolsSection.waitFor({ state: 'visible', timeout: 5000 });

    const toolsToggle = serverCard.locator('.tools-toggle');

    // Verify tools toggle shows tool count
    await expect(toolsToggle).toContainText('Tools');

    // Expand the tools section
    await mcpHelpers.expandServerTools(serverName);

    // Verify tools list is visible
    const toolsList = serverCard.locator('.tools-list');
    await expect(toolsList).toBeVisible();

    // Verify at least one tool is shown
    const tools = toolsList.locator('.tool-item');
    const toolCount = await tools.count();
    expect(toolCount).toBeGreaterThan(0);

    // Verify tool structure
    const firstTool = tools.first();
    await expect(firstTool.locator('.tool-name')).toBeVisible();
  });

  test('should remove an MCP server', async ({ page, mcpHelpers }) => {
    const serverName = `test-server-${Date.now()}`;

    await mcpHelpers.addServer(serverName, 'npx', '-y, @modelcontextprotocol/server-filesystem, /tmp');

    // Verify server is present
    let serverCard = page.locator(`.server-card:has-text("${serverName}")`);
    await expect(serverCard).toBeVisible();

    // Remove the server
    await mcpHelpers.removeServer(serverName);

    // Verify server is removed
    serverCard = page.locator(`.server-card:has-text("${serverName}")`);
    await expect(serverCard).not.toBeVisible();

    // Verify success toast
    await expect(page.locator('.toast-success')).toBeVisible();
  });

  test('should validate required fields when adding server', async ({ page }) => {
    await page.click('button:has-text("Add Server")');

    // Wait for form to be visible
    await page.waitForSelector('.add-server-form');

    // Fill only the name field, leave command empty to trigger validation
    await page.fill('#server-name', 'test-incomplete-server');

    // Try to submit - this should trigger validation since command is required
    // We need to use evaluate to bypass HTML5 validation and trigger the React validation
    await page.evaluate(() => {
      const form = document.querySelector('.add-server-form') as HTMLFormElement;
      if (form) {
        // Temporarily remove required attributes to bypass browser validation
        const inputs = form.querySelectorAll('input[required]');
        inputs.forEach(input => input.removeAttribute('required'));
        // Submit the form
        form.requestSubmit();
      }
    });

    // Should show validation or error toast from the app
    const toast = page.locator('.toast-error');
    await expect(toast).toBeVisible();
    await expect(toast).toContainText('required');
  });

  test('should cancel adding a server', async ({ page }) => {
    await page.click('button:has-text("Add Server")');

    // Verify form is visible
    const form = page.locator('.add-server-form');
    await expect(form).toBeVisible();

    // Fill some data
    await page.fill('#server-name', 'test-server');

    // Click cancel
    await page.click('button.cancel-btn');

    // Verify form is hidden
    await expect(form).not.toBeVisible();
  });

  test('should show error state for failed server connection', async ({ page, mcpHelpers }) => {
    const serverName = `test-server-${Date.now()}`;

    // Add a server with an invalid command
    await mcpHelpers.addServer(serverName, '/invalid/command/path', '');

    // Wait for status to update
    await page.waitForTimeout(2000);

    const serverCard = page.locator(`.server-card:has-text("${serverName}")`);

    // Should show error status
    const statusBadge = serverCard.locator('.status-badge');
    await expect(statusBadge).toContainText('Error');

    // May show error details
    const errorDetail = serverCard.locator('.error-detail');
    if (await errorDetail.isVisible()) {
      await expect(errorDetail).toBeVisible();
    }
  });

  test('should support adding multiple servers', async ({ page, mcpHelpers }) => {
    const servers = [
      { name: `server-1-${Date.now()}`, command: 'npx' },
      { name: `server-2-${Date.now()}`, command: 'npx' },
    ];

    for (const server of servers) {
      await mcpHelpers.addServer(server.name, server.command, '-y, @modelcontextprotocol/server-filesystem, /tmp');
    }

    // Verify both servers are visible
    for (const server of servers) {
      const serverCard = page.locator(`.server-card:has-text("${server.name}")`);
      await expect(serverCard).toBeVisible();
    }

    // Verify server count
    const allServers = page.locator('.server-card');
    const count = await allServers.count();
    expect(count).toBeGreaterThanOrEqual(2);
  });

  test('should close settings', async ({ page }) => {
    // Close button should be available (depends on implementation)
    const closeButton = page.locator('button[title="Close"], .close-btn, button:has-text("Close")');

    if (await closeButton.isVisible()) {
      await closeButton.click();

      // Verify settings panel is hidden
      const mcpSettings = page.locator('.mcp-settings');
      await expect(mcpSettings).not.toBeVisible();
    }
  });
});
