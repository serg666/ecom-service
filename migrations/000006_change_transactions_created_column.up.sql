ALTER TABLE ONLY public.transactions ALTER COLUMN created SET DEFAULT (now() AT TIME ZONE 'utc');

UPDATE public.transactions SET created = (created - interval '3 hours');
