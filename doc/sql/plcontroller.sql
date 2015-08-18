CREATE TABLE `vantage_point` (
  `ip` int(10) unsigned NOT NULL DEFAULT '0',
  `controller` int(10) unsigned DEFAULT NULL,
  `hostname` varchar(255) NOT NULL,
  `site` varchar(255) NOT NULL DEFAULT '',
  `timestamp` tinyint(1) NOT NULL DEFAULT '0',
  `record_route` tinyint(1) NOT NULL DEFAULT '0',
  `can_spoof` tinyint(1) NOT NULL DEFAULT '0',
  `receive_spoof` tinyint(1) NOT NULL DEFAULT '0',
  `port` int(11) NOT NULL DEFAULT '22',
  `last_updated` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `spoof_checked` datetime DEFAULT NULL,
  `last_health_check` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`ip`),
  UNIQUE KEY `ip_UNIQUE` (`ip`),
  UNIQUE KEY `hostname_UNIQUE` (`hostname`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;
