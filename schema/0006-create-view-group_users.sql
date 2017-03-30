CREATE
	VIEW `group_users` AS
    SELECT
        `u`.`group_id` AS `group_id`,
        `g`.`aspect` AS `aspect`,
        `g`.`group` AS `group`,
        `u`.`user` AS `user`
    FROM
        (`group_user` `u`
        JOIN `group` `g` ON ((`u`.`group_id` = `g`.`group_id`)))