-- Test API keys for development. DO NOT use in production.
--
-- Raw key 1: 27a8c227cbd77ac74510111056ac12d7146f869f97c972d192dcb35eafd5a393
-- Raw key 2: cfecc3d8dd29917014c5bc02ccffa8e1d21c56cb2c24aa0a2306816751ce1cdf
-- Raw key 3: 031bc44758498c47a0ca9f97b3707966641704afd26a28c215bcd3b952cc6f65

INSERT INTO api_keys (key_hash, key_prefix, name) VALUES
    ('f4590b88f3e1b3dd5ff2b69cd44f281cf09582348a1a7f47a6c33d1ddf9196ea', '27a8c227', 'test-key-1'),
    ('69d14ccfc8f616fffe55549f13f967dfad89361ef505e83e10cd271420f83742', 'cfecc3d8', 'test-key-2'),
    ('5d670dcaac8e7d9ce6537d5d4febd45cf1ce64640de2270d4adbf579a152d325', '031bc447', 'test-key-3');
