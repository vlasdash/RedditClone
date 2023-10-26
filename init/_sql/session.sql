DROP TABLE IF EXISTS `sessions`;
CREATE TABLE `sessions` (
                         `token` varchar(200) NOT NULL PRIMARY KEY,
                         `username` varchar(100) NOT NULL,
                         `user_id` int(11) UNSIGNED NOT NULL,
                         `expiration_date` varchar(100) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;