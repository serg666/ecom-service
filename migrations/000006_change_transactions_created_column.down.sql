ALTER TABLE ONLY public.transactions ALTER COLUMN created SET DEFAULT CURRENT_TIMESTAMP;

UPDATE public.transactions SET created = (created + interval '3 hours');
