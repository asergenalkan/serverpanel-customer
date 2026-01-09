1️⃣ En “resmi” yol: phpMyAdmin signon auth + minik PHP köprü

phpMyAdmin zaten bunun için özel bir mod veriyor:

$cfg['Servers'][$i]['auth_type']      = 'signon';
$cfg['Servers'][$i]['SignonSession']  = 'PMA_SIGNON_SESSION';
$cfg['Servers'][$i]['SignonURL']      = 'https://panel.domain.com/pma-sso-start';
$cfg['Servers'][$i]['SignonScript']   = '/var/www/pma_sso/signon.php';


Mantık şu:

Kullanıcı, senin Go paneline login oluyor (JWT vs. aynen devam).

Panelde “phpMyAdmin’e Git” butonuna basıyor.

Go backend şu route’u çalıştırıyor: /pma-sso-start

Kullanıcının hangi MySQL user/pass/db ile gireceğine karar veriyorsun.

Bu bilgiyi direkt göndermiyorsun, bir one-time token üretiyorsun (UUID vs).

Bu token → Redis / DB’de çok kısa süreli (30 sn – 1 dk) saklanıyor:

token → {db_user, db_pass, db_host, db_name}

Kullanıcıyı şu adrese redirect ediyorsun:
https://server/phpmyadmin/index.php?token=XYZ

phpMyAdmin auth_type = signon olduğu için her request’te SignonScript’i (signon.php) çağırıyor.

signon.php içi (basit haliyle):

<?php
session_name('PMA_SIGNON_SESSION');
session_start();

// 1) Token'ı al
$token = $_GET['token'] ?? null;
if (!$token) {
    // Token yoksa, phpMyAdmin login ekranına düşsün
    return;
}

// 2) Go backend'e istek at, token'ı doğrula
// Örn: http://127.0.0.1:9000/internal/pma-token/XYZ
// Gelen response: { "user": "dbuser", "password": "secret", "host": "localhost" }

$resp = file_get_contents("http://127.0.0.1:9000/internal/pma-token/" . urlencode($token));
$data = json_decode($resp, true);
if (!$data || empty($data['user'])) {
    return;
}

// 3) phpMyAdmin'in beklediği değişkenleri doldur
$_SESSION['PMA_single_signon_user']      = $data['user'];
$_SESSION['PMA_single_signon_password']  = $data['password'];
$_SESSION['PMA_single_signon_host']      = $data['host'] ?? 'localhost';
// varsa port, dbname vs ayarlarsın

// 4) Token'ı bir daha kullanılamasın diye Go backend'e "consume" ettir
file_get_contents("http://127.0.0.1:9000/internal/pma-token-consume/" . urlencode($token));


Bu akışta:

Kimlik sağlayıcı Go (panelin değişmiyor).

PHP sadece küçük bir “adapter”:

Go’dan gelen token’ı alıyor

phpMyAdmin’in istediği PMA_single_signon_* session değişkenlerini dolduruyor

phpMyAdmin tamamen kendi resmi signon flow’unu kullanıyor. Yani “resmi, dokümantasyonluk” çözüm.

Yani “ortada illa kocaman PHP SSO uygulaması olacak, panel de PHP’ye taşınacak” falan yok.
20–30 satırlık mini bir signon.php yeter.

Bu, gerçekten WHM/cPanel’e en benzeyen yöntem.