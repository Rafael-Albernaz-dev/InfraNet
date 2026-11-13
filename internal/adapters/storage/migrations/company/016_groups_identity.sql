CREATE TABLE device_groups (
 id TEXT PRIMARY KEY,
 parent_id TEXT NOT NULL DEFAULT '',
 name TEXT NOT NULL
);
CREATE UNIQUE INDEX device_group_siblings ON device_groups(parent_id, name COLLATE NOCASE);
ALTER TABLE devices ADD COLUMN group_id TEXT NOT NULL DEFAULT '';
ALTER TABLE devices ADD COLUMN identity TEXT NOT NULL DEFAULT '';
UPDATE devices SET identity=COALESCE(json_extract(observation,'$.identity'),'');
UPDATE table_preferences SET body=json_remove(json_set(body,'$.columns',
 json(COALESCE((SELECT json_group_array(CASE value WHEN 'block' THEN 'group' ELSE value END)
 FROM json_each(body,'$.columns')),'[]'))),'$.filters.block')
WHERE table_id='equipment';
UPDATE table_preferences SET body=json_set(body,'$.sort.column','group')
WHERE table_id='equipment' AND json_extract(body,'$.sort.column')='block';
