DROP TABLE IF EXISTS `users`;
CREATE TABLE `users` (
                         `id` int(11) UNSIGNED NOT NULL PRIMARY KEY AUTO_INCREMENT,
                         `username` varchar(100) NOT NULL,
                         `password` varchar(100) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8;