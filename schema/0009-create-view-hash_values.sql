CREATE 
    VIEW `hash_values` AS
    SELECT 
        `hash`.`hash_id` AS `hash_id`,
        `hash`.`aspect` AS `aspect`,
        `hash`.`hash_name` AS `hash_name`,
        `hash_value`.`hash_key` AS `hash_key`,
        `hash_value`.`hash_value` AS `hash_value`
    FROM
        (`hash`
        JOIN `hash_value` ON ((`hash`.`hash_id` = `hash_value`.`hash_id`)))
