ALTER TABLE public.accounts ADD COLUMN name varchar(255);
ALTER TABLE public.accounts ADD COLUMN created_at TIMESTAMP default NOW();
