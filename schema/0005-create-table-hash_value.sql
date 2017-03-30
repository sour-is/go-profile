CREATE TABLE `hash_value` (
  `hash_id` int(11) NOT NULL,
  `hash_key` varchar(255) NOT NULL,
  `hash_value` varchar(2000) NOT NULL,
  PRIMARY KEY (`hash_id`,`hash_key`),
  CONSTRAINT `fk_hash_value`
    FOREIGN KEY (`hash_id`)
    REFERENCES `hash` (`hash_id`)
      ON DELETE CASCADE
      ON UPDATE NO ACTION
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin