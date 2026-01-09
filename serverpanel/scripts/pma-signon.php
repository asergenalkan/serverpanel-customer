<?php
/**
 * ServerPanel - phpMyAdmin Single Sign-On Script
 * Bu script, panel'den gelen token ile phpMyAdmin'e otomatik giriş sağlar
 */

session_name('SignonSession');
session_start();

// Token'ı al
$token = $_GET['token'] ?? '';

if (empty($token)) {
    die('Token gerekli');
}

// Token'ı decode et (base64 encoded JSON)
$decoded = base64_decode($token);
if ($decoded === false) {
    die('Geçersiz token');
}

$data = json_decode($decoded, true);
if (!$data || !isset($data['user']) || !isset($data['password']) || !isset($data['db'])) {
    die('Token parse hatası');
}

// Token süresini kontrol et (5 dakika)
if (isset($data['exp']) && time() > $data['exp']) {
    die('Token süresi dolmuş');
}

// phpMyAdmin session bilgilerini ayarla
$_SESSION['PMA_single_signon_user'] = $data['user'];
$_SESSION['PMA_single_signon_password'] = $data['password'];
$_SESSION['PMA_single_signon_host'] = 'localhost';

// phpMyAdmin'e yönlendir
$pmaUrl = '/phpmyadmin/index.php';
if (!empty($data['db'])) {
    $pmaUrl .= '?db=' . urlencode($data['db']);
}

header('Location: ' . $pmaUrl);
exit;
