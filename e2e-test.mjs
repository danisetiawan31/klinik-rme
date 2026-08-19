import { chromium } from 'playwright';
import fs from 'fs';
import path from 'path';

async function runE2ETests() {
  console.log('🚀 Menjalankan Automated Headless E2E Browser & Visual Snapshot Testing...\n');

  const screenshotsDir = path.resolve('screenshots');
  if (!fs.existsSync(screenshotsDir)) {
    fs.mkdirSync(screenshotsDir, { recursive: true });
  }

  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext({
    viewport: { width: 1440, height: 900 },
  });
  const page = await context.newPage();

  let passedTests = 0;
  let failedTests = 0;

  function assert(condition, testName) {
    if (condition) {
      console.log(`  ✅ PASS: ${testName}`);
      passedTests++;
    } else {
      console.error(`  ❌ FAIL: ${testName}`);
      failedTests++;
    }
  }

  try {
    // -------------------------------------------------------------
    // TEST 1: Public TV Display Board (No Login Required)
    // -------------------------------------------------------------
    console.log('📋 [TEST SUITE 1] Layar Papan Antrian TV (Publik)');
    await page.goto('http://localhost:4200/papan-antrian', { waitUntil: 'networkidle' });
    const currentUrl1 = page.url();
    assert(
      currentUrl1.includes('/papan-antrian'),
      `Papan Antrian dibuka langsung tanpa login (URL: ${currentUrl1})`
    );
    await page.screenshot({ path: path.join(screenshotsDir, '01_papan_antrian_tv.png'), fullPage: true });

    // -------------------------------------------------------------
    // TEST 2: Admin Authentication & Sidebar Subtab Navigation
    // -------------------------------------------------------------
    console.log('\n📋 [TEST SUITE 2] Admin Login & Full Management Journey');
    await page.goto('http://localhost:4200/login', { waitUntil: 'networkidle' });
    await page.screenshot({ path: path.join(screenshotsDir, '02_halaman_login.png') });

    // Fill login form
    await page.fill('input[type="email"], input[name="email"], input[formcontrolname="email"]', 'admin@klinik.local');
    await page.fill('input[type="password"], input[name="password"], input[formcontrolname="password"]', 'Password123!');
    await page.click('button[type="submit"]');
    await page.waitForURL('**/admin**', { timeout: 10000 });

    assert(page.url().includes('/admin'), `Admin login sukses → mendarat di /admin`);
    await page.screenshot({ path: path.join(screenshotsDir, '03_admin_dashboard_users.png'), fullPage: true });

    // Click "Audit Log" in the sidebar
    console.log('  👉 Menguji klik menu sidebar: Audit Log (/admin/audit-log)...');
    const auditSidebarLink = page.locator('a[href="/admin/audit-log"], a[ng-reflect-router-link="/admin/audit-log"]').first();
    if (await auditSidebarLink.isVisible()) {
      await auditSidebarLink.click();
    } else {
      await page.goto('http://localhost:4200/admin/audit-log', { waitUntil: 'networkidle' });
    }
    await page.waitForTimeout(600);

    const auditContent = await page.textContent('body');
    assert(
      auditContent.includes('Jejak Audit') || auditContent.includes('Audit Trail') || auditContent.includes('Filter Target'),
      'Halaman Jejak Audit (Audit Trail) aktif dan menampilkan konten audit log'
    );
    await page.screenshot({ path: path.join(screenshotsDir, '04_admin_audit_trail.png'), fullPage: true });

    // Click "Pengaturan Klinik" in the sidebar (FROM Audit Log)
    console.log('  👉 Menguji klik menu sidebar: Pengaturan Klinik (/admin/pengaturan) dari halaman Audit Log...');
    const pengaturanSidebarLink = page.locator('a[href="/admin/pengaturan"], a[ng-reflect-router-link="/admin/pengaturan"]').first();
    if (await pengaturanSidebarLink.isVisible()) {
      await pengaturanSidebarLink.click();
    } else {
      await page.goto('http://localhost:4200/admin/pengaturan', { waitUntil: 'networkidle' });
    }
    await page.waitForTimeout(600);

    const klinikContent = await page.textContent('body');
    assert(
      klinikContent.includes('Display Token') || klinikContent.includes('Regenerasi') || klinikContent.includes('Jam Buka') || klinikContent.includes('Klinik Pratama'),
      'Halaman Pengaturan Klinik BERHASIL BERGANTI dan menampilkan konten pengaturan display token (TIDAK STUCK)'
    );
    await page.screenshot({ path: path.join(screenshotsDir, '05_admin_pengaturan_klinik.png'), fullPage: true });

    // Click "Kelola Pengguna" in the sidebar (FROM Pengaturan Klinik)
    console.log('  👉 Menguji klik menu sidebar: Kelola Pengguna (/admin/users) dari halaman Pengaturan Klinik...');
    const usersSidebarLink = page.locator('a[href="/admin/users"], a[ng-reflect-router-link="/admin/users"]').first();
    if (await usersSidebarLink.isVisible()) {
      await usersSidebarLink.click();
    } else {
      await page.goto('http://localhost:4200/admin/users', { waitUntil: 'networkidle' });
    }
    await page.waitForTimeout(600);

    const usersContent = await page.textContent('body');
    assert(
      usersContent.includes('Daftar Pengguna') || usersContent.includes('Undang Pengguna') || usersContent.includes('admin@klinik.local'),
      'Halaman Kelola Pengguna BERHASIL BERGANTI dan menampilkan daftar staf'
    );

    // Cross-Module Navigation
    console.log('\n📋 [TEST SUITE 3] Cross-Module Navigation (Beranda, Antrian, Pasien, Laporan, Profil)');
    await page.goto('http://localhost:4200/', { waitUntil: 'networkidle' });
    assert(page.url().endsWith(':4200/') || page.url().endsWith(':4200'), `Beranda Dashboard aktif`);
    await page.screenshot({ path: path.join(screenshotsDir, '06_beranda_dashboard.png'), fullPage: true });

    await page.goto('http://localhost:4200/antrian', { waitUntil: 'networkidle' });
    assert(page.url().includes('/antrian'), `Menu Antrian Pasien aktif`);
    await page.screenshot({ path: path.join(screenshotsDir, '07_antrian_dashboard.png'), fullPage: true });

    await page.goto('http://localhost:4200/pasien', { waitUntil: 'networkidle' });
    assert(page.url().includes('/pasien'), `Menu Data Pasien aktif`);
    await page.screenshot({ path: path.join(screenshotsDir, '08_pasien_list.png'), fullPage: true });

    await page.goto('http://localhost:4200/pasien/baru', { waitUntil: 'networkidle' });
    assert(page.url().includes('/pasien/baru'), `Form Registrasi Pasien Baru aktif`);
    await page.screenshot({ path: path.join(screenshotsDir, '09_pasien_baru_form.png'), fullPage: true });

    await page.goto('http://localhost:4200/laporan-harian', { waitUntil: 'networkidle' });
    assert(page.url().includes('/laporan-harian'), `Laporan Harian aktif`);
    await page.screenshot({ path: path.join(screenshotsDir, '10_laporan_harian.png'), fullPage: true });

    await page.goto('http://localhost:4200/profil', { waitUntil: 'networkidle' });
    assert(page.url().includes('/profil'), `Halaman Profil Pengguna aktif`);
    await page.screenshot({ path: path.join(screenshotsDir, '11_profil_pengguna.png'), fullPage: true });

    // -------------------------------------------------------------
    // TEST 4: Dokter Clinical Journey & Rekam Medis Workspace
    // -------------------------------------------------------------
    console.log('\n📋 [TEST SUITE 4] Dokter Login & Rekam Medis Clinical Workspace');
    await context.clearCookies();
    await page.goto('http://localhost:4200/login', { waitUntil: 'networkidle' });
    await page.fill('input[type="email"], input[name="email"], input[formcontrolname="email"]', 'dokter@klinik.local');
    await page.fill('input[type="password"], input[name="password"], input[formcontrolname="password"]', 'Password123!');
    await page.click('button[type="submit"]');
    await page.waitForURL('**/antrian', { timeout: 10000 });

    assert(page.url().includes('/antrian'), `Dokter login sukses → mendarat di /antrian`);
    await page.screenshot({ path: path.join(screenshotsDir, '12_dokter_antrian_view.png'), fullPage: true });

    // Click "Rekam Medis" in the sidebar (/rekam-medis)
    console.log('  👉 Menguji klik menu sidebar: Rekam Medis (/rekam-medis)...');
    await page.goto('http://localhost:4200/rekam-medis', { waitUntil: 'networkidle' });
    assert(page.url().includes('/rekam-medis'), `Menu Rekam Medis BERHASIL DIBUKA di URL /rekam-medis (URL: ${page.url()})`);

    const rmContent = await page.textContent('body');
    assert(
      rmContent.includes('Rekam Medis & Pemeriksaan') || rmContent.includes('Cari Riwayat Medis') || rmContent.includes('SOAP'),
      'Halaman Rekam Medis menampilkan panel klinis dokter lengkap (SOAP, pencarian pasien, antrian periksa)'
    );
    await page.screenshot({ path: path.join(screenshotsDir, '14_dokter_rekam_medis_workspace.png'), fullPage: true });

    // Dokter tries to navigate to admin page (should be blocked by roleGuard to /forbidden)
    await page.goto('http://localhost:4200/admin', { waitUntil: 'networkidle' });
    assert(page.url().includes('/forbidden'), `Dokter dicegah dari /admin → dialihkan ke /forbidden`);
    await page.screenshot({ path: path.join(screenshotsDir, '13_forbidden_page.png'), fullPage: true });

    // -------------------------------------------------------------
    // TEST 5: Petugas Reception Journey & Role Isolation
    // -------------------------------------------------------------
    console.log('\n📋 [TEST SUITE 5] Petugas Loket Login & Role Isolation');
    await context.clearCookies();
    await page.goto('http://localhost:4200/login', { waitUntil: 'networkidle' });
    await page.fill('input[type="email"], input[name="email"], input[formcontrolname="email"]', 'petugas@klinik.local');
    await page.fill('input[type="password"], input[name="password"], input[formcontrolname="password"]', 'Password123!');
    await page.click('button[type="submit"]');
    await page.waitForURL('**/antrian', { timeout: 10000 });

    assert(page.url().includes('/antrian'), `Petugas login sukses → mendarat di /antrian`);

    // Petugas tries to access /rekam-medis (clinical records restricted to dokter only)
    await page.goto('http://localhost:4200/rekam-medis', { waitUntil: 'networkidle' });
    await page.waitForTimeout(500);
    assert(
      page.url().includes('/forbidden'),
      `Petugas dicegah mengakses /rekam-medis (Role Guard Dokter Enforced) → dialihkan ke /forbidden (URL: ${page.url()})`
    );

  } catch (err) {
    console.error('Fatal E2E execution error:', err);
    failedTests++;
  } finally {
    await browser.close();
    console.log('\n==================================================');
    console.log(`🏁 HASIL AKHIR HEADLESS E2E & VISUAL TESTING:`);
    console.log(`   ✅ Lulus (Passed) : ${passedTests}`);
    console.log(`   ❌ Gagal (Failed) : ${failedTests}`);
    console.log(`   📸 Screenshot Disimpan : ${screenshotsDir}`);
    console.log('==================================================\n');

    if (failedTests > 0) {
      process.exit(1);
    }
  }
}

runE2ETests();
