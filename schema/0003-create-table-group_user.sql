CREATE TABLE `group_user` (
  `group_id` int(11) NOT NULL,
  `user` varchar(45) NOT NULL,
  PRIMARY KEY (`group_id`,`user`),
  KEY `user` (`user`),
  CONSTRAINT `fk_group_user`
  FOREIGN KEY (`group_id`)
  REFERENCES `group` (`group_id`)
    ON DELETE CASCADE
    ON UPDATE NO ACTION
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin