CREATE
    VIEW `user_roles` AS
    SELECT DISTINCT
        `u`.`user` AS `user`,
        REPLACE(`r`.`aspect`,
            ':user',
            `u`.`user`) AS `aspect`,
        REPLACE(`r`.`role`, ':user', `u`.`user`) AS `role`
    FROM
        (`group_users` `u`
        JOIN `group_roles` `r` ON (((`u`.`aspect` = `r`.`assign_aspect`)
            AND (`u`.`group` = `r`.`assign_group`))))