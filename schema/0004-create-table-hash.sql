CREATE TABLE `hash` (
  `hash_id` int(11) NOT NULL AUTO_INCREMENT,
  `aspect` varchar(255) NOT NULL,
  `hash_name` varchar(255) NOT NULL,
  PRIMARY KEY (`hash_id`),
  KEY `hash` (`aspect`,`hash_name`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8 COLLATE=utf8_bin
