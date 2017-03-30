CREATE TABLE `group_role` (
  `group_id` int(11) DEFAULT NULL,
  `aspect` varchar(255) NOT NULL,
  `role` varchar(255) NOT NULL,
  KEY `aspect` (`aspect`,`role`),
  KEY `fk_group_role` (`group_id`),
  CONSTRAINT `fk_group_role`
  FOREIGN KEY (`group_id`)
  REFERENCES `group` (`group_id`)
    ON DELETE CASCADE
    ON UPDATE NO ACTION
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin