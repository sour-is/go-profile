CREATE OR REPLACE
  VIEW `aspects` AS
    SELECT
        `group`.`aspect` AS `aspect`
    FROM
        `group`
    UNION SELECT
        `hash`.`aspect` AS `aspect`
    FROM
        `hash`