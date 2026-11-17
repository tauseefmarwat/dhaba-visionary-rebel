INSERT INTO categories(name,description) VALUES
 ('Karahi & Handi','Traditional karahi and handi dishes'),
 ('BBQ','Grilled favourites'),
 ('Rice','Rice and biryani specials'),
 ('Breads','Freshly baked breads'),
 ('Drinks','Cold drinks and refreshments')
ON CONFLICT(name) DO NOTHING;
INSERT INTO menu_items(category_id,name,description,price,image_url) VALUES
 ((SELECT id FROM categories WHERE name='Karahi & Handi'),'Chicken Karahi','Traditional tomato-based chicken karahi, served for sharing',1450,'https://images.unsplash.com/photo-1603894584373-5ac82b2ae398?w=800'),
 ((SELECT id FROM categories WHERE name='Karahi & Handi'),'Chicken Handi','Creamy handi with aromatic spices',1650,'https://images.unsplash.com/photo-1565557623262-b51c2513a641?w=800'),
 ((SELECT id FROM categories WHERE name='BBQ'),'Chicken Tikka','Char-grilled chicken with house spices',650,'https://images.unsplash.com/photo-1599487488170-d11ec9c172f0?w=800'),
 ((SELECT id FROM categories WHERE name='BBQ'),'Seekh Kebab','Four juicy beef seekh kebabs',750,'https://images.unsplash.com/photo-1529042410759-befb1204b468?w=800'),
 ((SELECT id FROM categories WHERE name='Rice'),'Chicken Biryani','Fragrant basmati rice with chicken',420,'https://images.unsplash.com/photo-1563379091339-03246963d96c?w=800'),
 ((SELECT id FROM categories WHERE name='Rice'),'Mutton Pulao','Yakhni rice with tender mutton',580,'https://images.unsplash.com/photo-1512058564366-18510be2db19?w=800'),
 ((SELECT id FROM categories WHERE name='Breads'),'Tandoori Naan','Fresh tandoor-baked naan',60,'https://images.unsplash.com/photo-1601050690597-df0568f70950?w=800'),
 ((SELECT id FROM categories WHERE name='Breads'),'Garlic Naan','Naan brushed with garlic butter',140,'https://images.unsplash.com/photo-1610192244261-3f33de3f55e4?w=800'),
 ((SELECT id FROM categories WHERE name='Drinks'),'Mint Margarita','Refreshing mint and lemon cooler',250,'https://images.unsplash.com/photo-1544145945-f90425340c7e?w=800'),
 ((SELECT id FROM categories WHERE name='Drinks'),'Lassi','Chilled traditional sweet yogurt drink',220,'https://images.unsplash.com/photo-1570197788417-0e82375c9371?w=800')
ON CONFLICT DO NOTHING;
