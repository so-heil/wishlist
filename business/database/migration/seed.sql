INSERT INTO "user" (email, username, password_hash, name, created_at)
    VALUES
        ('hosein@hotmail.com', 'hoseinzz', '0000', 'Hossein', '2023-03-24 00:00:00'),
        ('parisa@gmail.com', 'parisa01', '0000', 'Parisa', '2023-03-24 00:00:00');

INSERT INTO "wishlist" (name, description, created_at, updated_at, owner_id)
    VALUES
        ('Birthday Wishlist', null, '2023-03-24 00:00:00', '2023-03-24 00:00:00', 1),
        ('WANT', 'Things i really line', '2023-03-24 00:00:00', '2023-03-24 00:00:00', 1),
        ('Just Beautiful', null, '2023-03-24 00:00:00', '2023-03-24 00:00:00', 2);

INSERT INTO "product" (name, description, created_at, updated_at, image_url, price, price_updated_at, wishlist_id)
    VALUES
        ('Introduction to Algorithms, fourth edition 4th Edition',
         'A comprehensive update of the leading algorithms text, with new material on matchings in bipartite graphs, online algorithms, machine learning, and other topics.',
         '2023-03-24 00:00:00',
         '2023-03-24 00:00:00',
         null,
         '101$',
         '2023-03-24 00:00:00',
         2),
        ('Art of Computer Programming, The, Volumes 1-4B, Boxed Set (Art of Computer Programming, 1-4) 1st Edition',
         'Now Includes the Long-Anticipated Volume 4B!',
         '2023-03-24 00:00:00',
         '2023-03-24 00:00:00',
         null,
         '180.94$',
         '2023-03-24 00:00:00',
         2),
        ('Ring Sizer Adjuster for Loose Rings - 12 Pack, 2 Sizes for Different Band Widths',
         null,
         '2023-03-24 00:00:00',
         '2023-03-24 00:00:00',
         null,
         '14.99$',
         '2023-03-24 00:00:00',
         3);
