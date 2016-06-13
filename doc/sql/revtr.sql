-- MySQL dump 10.13  Distrib 5.6.30-76.3, for debian-linux-gnu (x86_64)
--
-- Host: localhost    Database: revtr
-- ------------------------------------------------------
-- Server version	5.6.30-76.3-log

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

--
-- Table structure for table `adjacencies`
--

DROP TABLE IF EXISTS `adjacencies`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `adjacencies` (
  `ip1` int(10) unsigned NOT NULL DEFAULT '0',
  `ip2` int(10) unsigned NOT NULL DEFAULT '0',
  `cnt` int(10) unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`ip1`,`ip2`),
  KEY `address2` (`ip2`)
) ENGINE=MyISAM DEFAULT CHARSET=latin1;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `adjacencies_to_dest`
--

DROP TABLE IF EXISTS `adjacencies_to_dest`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `adjacencies_to_dest` (
  `dest24` mediumint(8) unsigned NOT NULL DEFAULT '0',
  `address` int(10) unsigned NOT NULL DEFAULT '0',
  `adjacent` int(10) unsigned NOT NULL DEFAULT '0',
  `cnt` int(10) unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`address`,`dest24`,`adjacent`)
) ENGINE=MyISAM DEFAULT CHARSET=latin1;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `batch`
--

DROP TABLE IF EXISTS `batch`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `batch` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` int(10) unsigned NOT NULL,
  `created` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=2603 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `batch_revtr`
--

DROP TABLE IF EXISTS `batch_revtr`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `batch_revtr` (
  `batch_id` int(10) unsigned NOT NULL,
  `revtr_id` int(10) unsigned NOT NULL,
  KEY `fk_batch_revtr_2_idx` (`revtr_id`),
  KEY `fk_batch_revtr_batch_id` (`batch_id`),
  CONSTRAINT `fk_batch_revtr_2` FOREIGN KEY (`revtr_id`) REFERENCES `reverse_traceroutes` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION,
  CONSTRAINT `fk_batch_revtr_batch_id` FOREIGN KEY (`batch_id`) REFERENCES `batch` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `hop_types`
--

DROP TABLE IF EXISTS `hop_types`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `hop_types` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `type` varchar(45) NOT NULL DEFAULT '',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=10 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `ip_aliases`
--

DROP TABLE IF EXISTS `ip_aliases`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `ip_aliases` (
  `cluster_id` int(11) DEFAULT NULL,
  `ip_address` int(10) unsigned NOT NULL,
  PRIMARY KEY (`ip_address`),
  KEY `cluster_idx` (`cluster_id`)
) ENGINE=MyISAM DEFAULT CHARSET=latin1;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Temporary view structure for view `m-lab_revtrs`
--

DROP TABLE IF EXISTS `m-lab_revtrs`;
/*!50001 DROP VIEW IF EXISTS `m-lab_revtrs`*/;
SET @saved_cs_client     = @@character_set_client;
SET character_set_client = utf8;
/*!50001 CREATE VIEW `m-lab_revtrs` AS SELECT 
 1 AS `dst`,
 1 AS `src`,
 1 AS `date`,
 1 AS `hop1`,
 1 AS `hop2`,
 1 AS `hop3`,
 1 AS `hop4`,
 1 AS `hop5`,
 1 AS `hop6`,
 1 AS `hop7`,
 1 AS `hop8`,
 1 AS `hop9`,
 1 AS `hop10`,
 1 AS `hop11`,
 1 AS `hop12`,
 1 AS `hop13`,
 1 AS `hop14`,
 1 AS `hop15`,
 1 AS `hop16`,
 1 AS `hop17`,
 1 AS `hop18`,
 1 AS `hop19`,
 1 AS `hop20`,
 1 AS `hop21`,
 1 AS `hop22`,
 1 AS `hop23`,
 1 AS `hop24`,
 1 AS `hop25`,
 1 AS `hop26`,
 1 AS `hop27`,
 1 AS `hop28`,
 1 AS `hop29`,
 1 AS `hop30`,
 1 AS `type1`,
 1 AS `type2`,
 1 AS `type3`,
 1 AS `type4`,
 1 AS `type5`,
 1 AS `type6`,
 1 AS `type7`,
 1 AS `type8`,
 1 AS `type9`,
 1 AS `type10`,
 1 AS `type11`,
 1 AS `type12`,
 1 AS `type13`,
 1 AS `type14`,
 1 AS `type15`,
 1 AS `type16`,
 1 AS `type17`,
 1 AS `type18`,
 1 AS `type19`,
 1 AS `type20`,
 1 AS `type21`,
 1 AS `type22`,
 1 AS `type23`,
 1 AS `type24`,
 1 AS `type25`,
 1 AS `type26`,
 1 AS `type27`,
 1 AS `type28`,
 1 AS `type29`,
 1 AS `type30`,
 1 AS `runtime`,
 1 AS `rr_issued_count`,
 1 AS `ts_issued_count`*/;
SET character_set_client = @saved_cs_client;

--
-- Table structure for table `reverse_traceroute_hops`
--

DROP TABLE IF EXISTS `reverse_traceroute_hops`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `reverse_traceroute_hops` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `reverse_traceroute_id` int(10) unsigned NOT NULL,
  `hop` int(10) unsigned NOT NULL,
  `hop_type` int(10) unsigned NOT NULL,
  `order` int(10) unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  KEY `fk_reverse_traceroute_hops_2_idx` (`hop_type`),
  KEY `fk_reverse_traceroute_hops_1_idx` (`reverse_traceroute_id`),
  CONSTRAINT `fk_reverse_traceroute_hops_1` FOREIGN KEY (`reverse_traceroute_id`) REFERENCES `reverse_traceroutes` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION,
  CONSTRAINT `fk_reverse_traceroute_hops_2` FOREIGN KEY (`hop_type`) REFERENCES `hop_types` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION
) ENGINE=InnoDB AUTO_INCREMENT=2224650 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `reverse_traceroutes`
--

DROP TABLE IF EXISTS `reverse_traceroutes`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `reverse_traceroutes` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `src` int(10) unsigned NOT NULL,
  `dst` int(10) unsigned NOT NULL,
  `runtime` bigint(20) NOT NULL DEFAULT '0',
  `rr_issued` int(10) unsigned NOT NULL DEFAULT '0',
  `ts_issued` int(10) unsigned NOT NULL DEFAULT '0',
  `stop_reason` varchar(45) NOT NULL DEFAULT '',
  `status` varchar(45) NOT NULL DEFAULT 'RUNNING',
  `fail_reason` varchar(255) DEFAULT '',
  `date` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `index2` (`src`,`dst`)
) ENGINE=InnoDB AUTO_INCREMENT=211838 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `users`
--

DROP TABLE IF EXISTS `users`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `users` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL DEFAULT '',
  `email` varchar(255) NOT NULL DEFAULT '',
  `max` int(10) unsigned NOT NULL DEFAULT '0',
  `delay` int(10) unsigned NOT NULL DEFAULT '0',
  `key` varchar(100) NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  KEY `index2` (`key`)
) ENGINE=InnoDB AUTO_INCREMENT=9 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Final view structure for view `m-lab_revtrs`
--

/*!50001 DROP VIEW IF EXISTS `m-lab_revtrs`*/;
/*!50001 SET @saved_cs_client          = @@character_set_client */;
/*!50001 SET @saved_cs_results         = @@character_set_results */;
/*!50001 SET @saved_col_connection     = @@collation_connection */;
/*!50001 SET character_set_client      = utf8 */;
/*!50001 SET character_set_results     = utf8 */;
/*!50001 SET collation_connection      = utf8_general_ci */;
/*!50001 CREATE ALGORITHM=UNDEFINED */
/*!50013 DEFINER=`root`@`129.10.113.189` SQL SECURITY DEFINER */
/*!50001 VIEW `m-lab_revtrs` AS select `rt`.`dst` AS `dst`,`rt`.`src` AS `src`,`rt`.`date` AS `date`,max((case when (`rth`.`order` = 0) then `rth`.`hop` else 0 end)) AS `hop1`,max((case when (`rth`.`order` = 1) then `rth`.`hop` else 0 end)) AS `hop2`,max((case when (`rth`.`order` = 2) then `rth`.`hop` else 0 end)) AS `hop3`,max((case when (`rth`.`order` = 3) then `rth`.`hop` else 0 end)) AS `hop4`,max((case when (`rth`.`order` = 4) then `rth`.`hop` else 0 end)) AS `hop5`,max((case when (`rth`.`order` = 5) then `rth`.`hop` else 0 end)) AS `hop6`,max((case when (`rth`.`order` = 6) then `rth`.`hop` else 0 end)) AS `hop7`,max((case when (`rth`.`order` = 7) then `rth`.`hop` else 0 end)) AS `hop8`,max((case when (`rth`.`order` = 8) then `rth`.`hop` else 0 end)) AS `hop9`,max((case when (`rth`.`order` = 9) then `rth`.`hop` else 0 end)) AS `hop10`,max((case when (`rth`.`order` = 10) then `rth`.`hop` else 0 end)) AS `hop11`,max((case when (`rth`.`order` = 11) then `rth`.`hop` else 0 end)) AS `hop12`,max((case when (`rth`.`order` = 12) then `rth`.`hop` else 0 end)) AS `hop13`,max((case when (`rth`.`order` = 13) then `rth`.`hop` else 0 end)) AS `hop14`,max((case when (`rth`.`order` = 14) then `rth`.`hop` else 0 end)) AS `hop15`,max((case when (`rth`.`order` = 15) then `rth`.`hop` else 0 end)) AS `hop16`,max((case when (`rth`.`order` = 16) then `rth`.`hop` else 0 end)) AS `hop17`,max((case when (`rth`.`order` = 17) then `rth`.`hop` else 0 end)) AS `hop18`,max((case when (`rth`.`order` = 18) then `rth`.`hop` else 0 end)) AS `hop19`,max((case when (`rth`.`order` = 19) then `rth`.`hop` else 0 end)) AS `hop20`,max((case when (`rth`.`order` = 20) then `rth`.`hop` else 0 end)) AS `hop21`,max((case when (`rth`.`order` = 21) then `rth`.`hop` else 0 end)) AS `hop22`,max((case when (`rth`.`order` = 22) then `rth`.`hop` else 0 end)) AS `hop23`,max((case when (`rth`.`order` = 23) then `rth`.`hop` else 0 end)) AS `hop24`,max((case when (`rth`.`order` = 24) then `rth`.`hop` else 0 end)) AS `hop25`,max((case when (`rth`.`order` = 25) then `rth`.`hop` else 0 end)) AS `hop26`,max((case when (`rth`.`order` = 26) then `rth`.`hop` else 0 end)) AS `hop27`,max((case when (`rth`.`order` = 27) then `rth`.`hop` else 0 end)) AS `hop28`,max((case when (`rth`.`order` = 28) then `rth`.`hop` else 0 end)) AS `hop29`,max((case when (`rth`.`order` = 29) then `rth`.`hop` else 0 end)) AS `hop30`,max((case when (`rth`.`order` = 0) then `rth`.`hop_type` else 0 end)) AS `type1`,max((case when (`rth`.`order` = 1) then `rth`.`hop_type` else 0 end)) AS `type2`,max((case when (`rth`.`order` = 2) then `rth`.`hop_type` else 0 end)) AS `type3`,max((case when (`rth`.`order` = 3) then `rth`.`hop_type` else 0 end)) AS `type4`,max((case when (`rth`.`order` = 4) then `rth`.`hop_type` else 0 end)) AS `type5`,max((case when (`rth`.`order` = 5) then `rth`.`hop_type` else 0 end)) AS `type6`,max((case when (`rth`.`order` = 6) then `rth`.`hop_type` else 0 end)) AS `type7`,max((case when (`rth`.`order` = 7) then `rth`.`hop_type` else 0 end)) AS `type8`,max((case when (`rth`.`order` = 8) then `rth`.`hop_type` else 0 end)) AS `type9`,max((case when (`rth`.`order` = 9) then `rth`.`hop_type` else 0 end)) AS `type10`,max((case when (`rth`.`order` = 10) then `rth`.`hop_type` else 0 end)) AS `type11`,max((case when (`rth`.`order` = 11) then `rth`.`hop_type` else 0 end)) AS `type12`,max((case when (`rth`.`order` = 12) then `rth`.`hop_type` else 0 end)) AS `type13`,max((case when (`rth`.`order` = 13) then `rth`.`hop_type` else 0 end)) AS `type14`,max((case when (`rth`.`order` = 14) then `rth`.`hop_type` else 0 end)) AS `type15`,max((case when (`rth`.`order` = 15) then `rth`.`hop_type` else 0 end)) AS `type16`,max((case when (`rth`.`order` = 16) then `rth`.`hop_type` else 0 end)) AS `type17`,max((case when (`rth`.`order` = 17) then `rth`.`hop_type` else 0 end)) AS `type18`,max((case when (`rth`.`order` = 18) then `rth`.`hop_type` else 0 end)) AS `type19`,max((case when (`rth`.`order` = 19) then `rth`.`hop_type` else 0 end)) AS `type20`,max((case when (`rth`.`order` = 20) then `rth`.`hop_type` else 0 end)) AS `type21`,max((case when (`rth`.`order` = 21) then `rth`.`hop_type` else 0 end)) AS `type22`,max((case when (`rth`.`order` = 22) then `rth`.`hop_type` else 0 end)) AS `type23`,max((case when (`rth`.`order` = 23) then `rth`.`hop_type` else 0 end)) AS `type24`,max((case when (`rth`.`order` = 24) then `rth`.`hop_type` else 0 end)) AS `type25`,max((case when (`rth`.`order` = 25) then `rth`.`hop_type` else 0 end)) AS `type26`,max((case when (`rth`.`order` = 26) then `rth`.`hop_type` else 0 end)) AS `type27`,max((case when (`rth`.`order` = 27) then `rth`.`hop_type` else 0 end)) AS `type28`,max((case when (`rth`.`order` = 28) then `rth`.`hop_type` else 0 end)) AS `type29`,max((case when (`rth`.`order` = 29) then `rth`.`hop_type` else 0 end)) AS `type30`,`rt`.`runtime` AS `runtime`,`rt`.`rr_issued` AS `rr_issued_count`,`rt`.`ts_issued` AS `ts_issued_count` from (`reverse_traceroutes` `rt` join `reverse_traceroute_hops` `rth` on((`rth`.`reverse_traceroute_id` = `rt`.`id`))) where (`rt`.`date` > (now() - interval 1 day)) group by `rt`.`src`,`rt`.`dst`,`rt`.`date` */;
/*!50001 SET character_set_client      = @saved_cs_client */;
/*!50001 SET character_set_results     = @saved_cs_results */;
/*!50001 SET collation_connection      = @saved_col_connection */;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2016-06-13 10:55:36
