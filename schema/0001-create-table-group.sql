CREATE TABLE `group` (
  `group_id` int(11) NOT NULL AUTO_INCREMENT,
  `aspect` varchar(255) NOT NULL,
  `group` varchar(255) NOT NULL,
  PRIMARY KEY (`group_id`),
  KEY `group` (`aspect`,`group`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8 COLLATE=utf8_bin
