<?php
/**
 * ServerPanel - phpMyAdmin Signon Configuration
 * Bu dosya /etc/phpmyadmin/conf.d/serverpanel.php olarak kopyalanmalı
 */

// Signon authentication ayarları
$cfg['Servers'][1]['auth_type'] = 'signon';
$cfg['Servers'][1]['SignonSession'] = 'SignonSession';
$cfg['Servers'][1]['SignonURL'] = '/pma-signon.php';
$cfg['Servers'][1]['LogoutURL'] = '/phpmyadmin/';
