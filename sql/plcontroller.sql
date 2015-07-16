CREATE TABLE `vantage_points` (
  `ip` int(10) unsigned NOT NULL,
  `controller` int(10) unsigned DEFAULT NULL,
  `hostname` varchar(255) NOT NULL,
  `timestamp` tinyint(1) NOT NULL,
  `record_route` tinyint(1) NOT NULL,
  `can_spoof` tinyint(1) NOT NULL,
  `active` tinyint(1) NOT NULL DEFAULT '0',
  `last_updated` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`ip`),
  UNIQUE KEY `ip_UNIQUE` (`ip`),
  UNIQUE KEY `hostname_UNIQUE` (`hostname`)
) ENGINE=InnoDB DEFAULT CHARSET=latin1;
