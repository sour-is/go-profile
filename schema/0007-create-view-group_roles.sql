CREATE
    VIEW `group_roles` AS
    SELECT
        `r`.`group_id` AS `group_id`,
        `r`.`aspect` AS `aspect`,
        `r`.`role` AS `role`,
        `g`.`aspect` AS `assign_aspect`,
        `g`.`group` AS `assign_group`
    FROM
        (`group_role` `r`
        JOIN `group` `g` ON ((`r`.`group_id` = `g`.`group_id`)))