INSERT INTO `group_role` (`group_id`, `aspect`, `role`)
VALUES
  ('1', '*', 'admin'),
  ('2', '*', 'inactive'),
  ('3', '*', '@:user'),
  ('3', '*', 'active'),
  ('3', '*', 'hash.write.@:user'),
  ('3', '@:user', 'owner')
