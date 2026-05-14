INSERT INTO game_systems (slug, canonical_name, badge_color, is_curated) VALUES
    -- D&D family
    ('dnd-5e-2014',            'D&D 5e (2014)',                '#E40712', TRUE),
    ('dnd-5e-2024',            'D&D 5e (2024)',                '#C2272D', TRUE),
    ('dnd-3-5e',               'D&D 3.5e',                     '#BE2020', TRUE),
    ('dnd-4e',                 'D&D 4e',                       '#9B1B1B', TRUE),
    ('adnd-2e',                'AD&D 2e',                      '#7A1515', TRUE),
    ('pathfinder-1e',          'Pathfinder 1e',                '#4E0E0E', TRUE),
    ('pathfinder-2e',          'Pathfinder 2e',                '#5D0000', TRUE),
    ('old-school-essentials',  'Old-School Essentials',        '#1A1A1A', TRUE),
    ('dungeon-crawl-classics', 'Dungeon Crawl Classics',       '#4B3621', TRUE),

    -- Horror / Investigation
    ('call-of-cthulhu-7e',     'Call of Cthulhu 7e',           '#1B5E20', TRUE),
    ('delta-green',            'Delta Green',                  '#2E7D32', TRUE),
    ('vaesen',                 'Vaesen',                       '#5D4037', TRUE),
    ('mothership',             'Mothership',                   '#263238', TRUE),

    -- Urban fantasy / Modern
    ('vampire-masquerade-5e',  'Vampire: The Masquerade 5th Ed', '#8B0000', TRUE),
    ('werewolf-apocalypse-5e', 'Werewolf: The Apocalypse 5th Ed','#6D4C00', TRUE),
    ('chronicles-of-darkness', 'Chronicles of Darkness',       '#37474F', TRUE),
    ('monster-of-the-week',    'Monster of the Week',          '#BF360C', TRUE),

    -- Sci-fi / Cyberpunk
    ('shadowrun-5e',           'Shadowrun 5e',                 '#00BFA5', TRUE),
    ('shadowrun-6e',           'Shadowrun 6e',                 '#00897B', TRUE),
    ('cyberpunk-2020',         'Cyberpunk 2020',               '#F9A825', TRUE),
    ('cyberpunk-red',          'Cyberpunk RED',                '#D50000', TRUE),
    ('starfinder',             'Starfinder',                   '#1565C0', TRUE),
    ('traveller',              'Traveller',                    '#0D47A1', TRUE),
    ('star-wars-ffg',          'Star Wars (FFG)',              '#FFD600', TRUE),

    -- Generic / Multisetting
    ('gurps',                  'GURPS',                        '#6A1B9A', TRUE),
    ('savage-worlds',          'Savage Worlds',                '#E65100', TRUE),
    ('fate-core',              'Fate Core',                    '#0277BD', TRUE),
    ('cypher-system',          'Cypher System',                '#00838F', TRUE),

    -- Narrative / Indie
    ('blades-in-the-dark',     'Blades in the Dark',           '#1A237E', TRUE),
    ('apocalypse-world',       'Apocalypse World',             '#880E4F', TRUE),
    ('dungeon-world',          'Dungeon World',                '#4A148C', TRUE),
    ('mork-borg',              'Mörk Borg',                    '#FFEB3B', TRUE),
    ('cy-borg',                'Cy_Borg',                      '#76FF03', TRUE),
    ('mcdm-rpg',               'MCDM RPG',                    '#FF6F00', TRUE),
    ('fiasco',                 'Fiasco',                       '#FF5722', TRUE),

    -- Other popular
    ('warhammer-fantasy-4e',   'Warhammer Fantasy Roleplay 4e','#B71C1C', TRUE),
    ('mutants-masterminds-3e', 'Mutants & Masterminds 3e',     '#2196F3', TRUE),
    ('numenera',               'Numenera',                     '#7C4DFF', TRUE),
    ('13th-age',               '13th Age',                     '#F57C00', TRUE),
    ('mythras',                'Mythras',                      '#795548', TRUE)
ON CONFLICT (slug) DO NOTHING;
