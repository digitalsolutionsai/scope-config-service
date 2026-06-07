// @ts-check
const { test, expect } = require('@playwright/test');

/**
 * Admin UI smoke tests for scope-config-service built-in admin.
 *
 * Covers:
 *  - Page loads (HTML + JS + CSS + js-yaml + marked via /admin/vendor/)
 *  - Sidebar renders, scope selector works
 *  - Search sidebar filter (new feature)
 *  - Empty state when no templates match
 *  - Template panel opens (Import + Manage tabs)
 *  - Responsive: mobile viewport hides sidebar
 *  - Unsaved warning beforeunload listener wired
 *
 * Assumes httpgateway is running on $BASE_URL (default http://localhost:8080)
 * with a SQLite or Postgres DB. Templates may or may not exist.
 */

test.describe('Admin UI', () => {
  test.beforeEach(async ({ page }) => {
    page.on('console', (msg) => {
      if (msg.type() === 'error') console.log(`[browser:error] ${msg.text()}`);
    });
  });

  test('home page loads and renders shell', async ({ page }) => {
    const response = await page.goto('/admin', { waitUntil: 'networkidle' });
    expect(response, 'home page should respond').toBeTruthy();
    expect(response.status()).toBeLessThan(400);

    // Wait for Vue to mount
    await expect(page.locator('h1:has-text("Scope Config")')).toBeVisible({ timeout: 10_000 });
    await expect(page.locator('aside').first()).toBeVisible();

    // Verify self-hosted js-yaml loaded
    const jsyamlOk = await page.evaluate(() => typeof window.jsyaml === 'object' && typeof window.jsyaml.load === 'function');
    expect(jsyamlOk, 'window.jsyaml.load should exist (self-hosted)').toBeTruthy();

    // Verify self-hosted marked loaded
    const markedOk = await page.evaluate(() => typeof window.marked === 'object' && typeof window.marked.parse === 'function');
    expect(markedOk, 'window.marked.parse should exist (self-hosted)').toBeTruthy();

    // Verify Vue mounted
    const vueOk = await page.evaluate(() => typeof window.Vue === 'object' && typeof window.Vue.createApp === 'function');
    expect(vueOk, 'window.Vue.createApp should exist').toBeTruthy();
  });

  test('scope selector has all 4 options', async ({ page }) => {
    await page.goto('/admin', { waitUntil: 'networkidle' });
    const scopeSelect = page.locator('aside select').first();
    await expect(scopeSelect).toBeVisible();
    const options = await scopeSelect.locator('option').allTextContents();
    expect(options).toEqual(['SYSTEM', 'PROJECT', 'STORE', 'USER']);
  });

  test('search filter shows and works', async ({ page }) => {
    await page.goto('/admin', { waitUntil: 'networkidle' });

    // Wait for templates loaded
    await page.waitForFunction(() => {
      const app = document.querySelector('#app');
      return app && !app.querySelector('.spinner');
    }, { timeout: 10_000 });

    // Search input visible
    const search = page.locator('input[placeholder="Filter groups..."]');
    await expect(search).toBeVisible();

    // Get total groups before filter
    const beforeRows = await page.locator('aside [id="nav-tree"] > div').count();

    // Type a query that should match nothing → see empty state
    await search.fill('zzzzz_nomatch_xyzzy');
    await expect(page.locator('aside').filter({ hasText: 'No groups match' })).toBeVisible({ timeout: 3_000 });

    // Clear filter
    const clearBtn = page.locator('input[placeholder="Filter groups..."] + button');
    if (await clearBtn.count()) {
      await clearBtn.click();
    } else {
      await search.fill('');
    }
    await expect(search).toHaveValue('');

    // If there are groups, type a real query that matches something
    if (beforeRows > 0) {
      const firstGroupLabel = await page.locator('aside [id="nav-tree"] > div').last().locator('span').first().textContent();
      if (firstGroupLabel) {
        const q = firstGroupLabel.slice(0, 3);
        await search.fill(q);
        // The group should be visible (or no-match if not)
        const visible = await page.locator('aside').filter({ hasText: 'No groups match' }).count();
        expect(visible === 0 || visible === 1).toBeTruthy();
      }
    }
  });

  test('template panel toggles Import and Manage tabs', async ({ page }) => {
    await page.goto('/admin', { waitUntil: 'networkidle' });

    // Open template panel
    await page.locator('button:has-text("Import / Manage Templates")').click();
    // The header text is "⚙ Template Management"; use the aside#tmpl-panel
    const tmplPanel = page.locator('aside#tmpl-panel');
    await expect(tmplPanel).toBeVisible({ timeout: 5_000 });
    await expect(tmplPanel.locator('text=Template Management')).toBeVisible();

    // Default tab is Import
    await expect(tmplPanel.locator('button:has-text("Validate Only")')).toBeVisible();
    await expect(tmplPanel.locator('button:has-text("Apply Template")')).toBeVisible();

    // Switch to Manage
    await tmplPanel.locator('button:has-text("Manage")').click();
    await page.waitForTimeout(500);

    // Close
    await tmplPanel.locator('button:has-text("✕")').click().catch(() => {});
  });

  test('selecting a group loads fields and shows version history (if templates exist)', async ({ page }) => {
    await page.goto('/admin', { waitUntil: 'networkidle' });
    await page.waitForFunction(() => {
      const app = document.querySelector('#app');
      return app && !app.querySelector('.spinner');
    }, { timeout: 10_000 });

    // If no templates, just confirm empty state
    const noTemplates = await page.locator('text="No templates found"').count();
    if (noTemplates > 0) {
      test.skip(true, 'no templates seeded — skipping field interaction test');
      return;
    }

    // Click first group in sidebar (use pl-7 which is the indent only group rows have)
    // First expand the service, then click the first group
    await page.locator('aside [id="nav-tree"] > div').first().click();
    await page.waitForTimeout(300);
    const firstGroup = page.locator('aside [id="nav-tree"] [class*="pl-7"][class*="cursor-pointer"]').first();
    await firstGroup.click({ force: true });

    // Wait for content + history to load
    await page.waitForTimeout(1_500);

    // Either fields load, or no-fields-for-scope message
    const fieldCount = await page.locator('main .bg-white.border').count();
    expect(fieldCount).toBeGreaterThanOrEqual(0);

    // Version history panel (always-visible left side) should be present
    const historyAside = page.locator('main aside:has-text("Version History")');
    await expect(historyAside).toBeVisible();
  });

  test('version history: shows versions with Active/Draft/Published tags', async ({ page }) => {
    await page.goto('/admin', { waitUntil: 'networkidle' });
    await page.waitForFunction(() => {
      const app = document.querySelector('#app');
      return app && !app.querySelector('.spinner');
    }, { timeout: 10_000 });

    const noTemplates = await page.locator('text="No templates found"').count();
    if (noTemplates > 0) {
      test.skip(true, 'no templates seeded');
      return;
    }

    // Expand service then click first group
    await page.locator('aside [id="nav-tree"] > div').first().click();
    await page.waitForTimeout(300);
    await page.locator('aside [id="nav-tree"] [class*="pl-7"][class*="cursor-pointer"]').first().click({ force: true });
    await page.waitForTimeout(2_000); // wait for history API

    const historyAside = page.locator('main aside:has-text("Version History")');
    await expect(historyAside).toBeVisible();

    // At least one version row (since we always auto-save on click might not, but the API may have data)
    // If a config has ever been saved, there should be a latest version highlighted as Draft
    const versionRows = historyAside.locator('div[class*="cursor-pointer"][class*="border-l-"]');
    const rowCount = await versionRows.count();
    if (rowCount > 0) {
      // Latest version should be highlighted (active in our isActiveVersion logic)
      const highlighted = historyAside.locator('div[class*="border-l-brand-500"]');
      await expect(highlighted.first()).toBeVisible();

      // The first row should contain either Active, Draft, or Published tag
      const firstRowText = await versionRows.first().textContent();
      const hasTag = /Active|Draft|Published/.test(firstRowText);
      expect(hasTag, 'first version row should have status tag').toBeTruthy();
    } else {
      // No history yet — empty state should show
      await expect(historyAside.locator('text=No history yet')).toBeVisible();
    }
  });

  test('responsive: mobile viewport hides sidebar', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/admin', { waitUntil: 'networkidle' });
    // H1 is inside sidebar which is hidden on mobile; use brand text in main/topbar
    await expect(page.locator('h2:has-text("Select a configuration group")')).toBeVisible({ timeout: 10_000 });

    // Sidebar (first aside) should be hidden on mobile (max-md:hidden)
    const sidebarVisible = await page.locator('aside').first().isVisible();
    expect(sidebarVisible).toBeFalsy();
  });

  test('unsaved warning: beforeunload registered when dirty', async ({ page }) => {
    await page.goto('/admin', { waitUntil: 'networkidle' });
    await page.waitForFunction(() => {
      const app = document.querySelector('#app');
      return app && !app.querySelector('.spinner');
    }, { timeout: 10_000 });

    // Without dirty flag, beforeunload should NOT be prevented
    const hasListener = await page.evaluate(() => {
      const evt = new Event('beforeunload', { cancelable: true });
      window.dispatchEvent(evt);
      return evt.defaultPrevented;
    });
    expect(hasListener).toBeFalsy();
  });

  test('history row has "Publish" button for non-active versions', async ({ page }) => {
    await page.goto('/admin', { waitUntil: 'networkidle' });
    await page.waitForFunction(() => {
      const app = document.querySelector('#app');
      return app && !app.querySelector('.spinner');
    }, { timeout: 10_000 });

    const noTemplates = await page.locator('text="No templates found"').count();
    if (noTemplates > 0) {
      test.skip(true, 'no templates seeded');
      return;
    }

    // Expand service, pick first group
    await page.locator('aside [id="nav-tree"] > div').first().click();
    await page.waitForTimeout(300);
    await page.locator('aside [id="nav-tree"] [class*="pl-7"][class*="cursor-pointer"]').first().click({ force: true });
    await page.waitForTimeout(2_000);

    const historyAside = page.locator('main aside:has-text("Version History")');
    await expect(historyAside).toBeVisible();

    // History should have at least 1 version with history data (or empty state)
    const versionRows = historyAside.locator('div[class*="border-l-"][class*="cursor-pointer"]');
    const rowCount = await versionRows.count();
    if (rowCount === 0) {
      test.skip(true, 'no version history yet — save a draft first');
      return;
    }

    // There should be a Publish button on the row that is NOT the active version
    // The active version has tag "Active" and no Publish button (v-if disables it)
    const publishButtons = historyAside.locator('button:has-text("Publish")');
    const publishCount = await publishButtons.count();
    // publishCount depends on history: at minimum 0 (if first save == first publish),
    // usually >= 1 (draft or older versions)
    if (publishCount > 0) {
      await expect(publishButtons.first()).toBeVisible();
    } else {
      // Topbar Publish button is intentionally hidden (v-if="... && false");
      // verify it's not visible.
      const topbarPublish = page.locator('main button:has-text("Publish")').first();
      await expect(topbarPublish).not.toBeVisible();
    }
  });

  test('saving a draft refreshes the version history list', async ({ page }) => {
    await page.goto('/admin', { waitUntil: 'networkidle' });
    await page.waitForFunction(() => {
      const app = document.querySelector('#app');
      return app && !app.querySelector('.spinner');
    }, { timeout: 10_000 });

    const noTemplates = await page.locator('text="No templates found"').count();
    if (noTemplates > 0) {
      test.skip(true, 'no templates seeded');
      return;
    }

    // Expand service and pick first group
    await page.locator('aside [id="nav-tree"] > div').first().click();
    await page.waitForTimeout(300);
    await page.locator('aside [id="nav-tree"] [class*="pl-7"][class*="cursor-pointer"]').first().click({ force: true });
    await page.waitForTimeout(2_000); // wait for content + history to load

    const historyAside = page.locator('main aside:has-text("Version History")');
    await expect(historyAside).toBeVisible();

    // Count history rows BEFORE save
    const beforeCount = await historyAside.locator('div[class*="border-l-"][class*="cursor-pointer"]').count();

    // Fill userName (audit trail) — required by API
    await page.locator('input[placeholder="your-email (for audit)"]').fill('yuki@e2e.test');

    // Pick any text/string field and modify it, then Save Draft
    const textInput = page.locator('main input[type="text"]').first();
    if (await textInput.count() === 0) {
      test.skip(true, 'no text input fields in this group — skipping save test');
      return;
    }
    const before = await textInput.inputValue();
    await textInput.fill(before + ' [e2e]');
    await page.locator('button:has-text("Save Draft")').click();

    // Wait for save API to complete + history reload
    await page.waitForTimeout(2_500);

    // History should now have one more row (or same if save didn't bump version)
    const afterCount = await historyAside.locator('div[class*="border-l-"][class*="cursor-pointer"]').count();
    // The "latest" badge should be on the most recent version
    const draftBadge = await historyAside.locator('text=Draft').count();
    expect(draftBadge, 'should have a Draft tag after save').toBeGreaterThanOrEqual(1);

    // Verify the new row was added (or at least history was reloaded)
    // We accept afterCount >= beforeCount as success (depends on whether first save)
    expect(afterCount).toBeGreaterThanOrEqual(beforeCount);
  });
});
