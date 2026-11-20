DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM permissions legacy
        JOIN permissions canonical
          ON canonical.code = regexp_replace(legacy.code, '\.([^.]+)$', ':\1')
        WHERE legacy.code ~ '\.[^.]+$'
          AND legacy.code !~ ':'
          AND legacy.id <> canonical.id
    ) THEN
        RAISE EXCEPTION
            'permission code normalization collision; resolve duplicate codes before applying this migration';
    END IF;
END $$;

UPDATE permissions
SET code = regexp_replace(code, '\.([^.]+)$', ':\1')
WHERE code ~ '\.[^.]+$'
  AND code !~ ':';
