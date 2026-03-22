import { expect, test } from '@playwright/test'

test('login and board page smoke', async ({ page }) => {
  const username = process.env.PW_ADMIN_USER ?? 'admin'
  const password = process.env.PW_ADMIN_PASSWORD ?? 'admin123'

  await page.goto('/login')
  await page.getByPlaceholder('admin').fill(username)
  await page.getByPlaceholder('******').fill(password)
  await page.getByRole('button', { name: '登录' }).click()

  await expect(page).toHaveURL(/\/board/)
  await expect(page.getByText('Agent Coder')).toBeVisible()
})
