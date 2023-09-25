ALTER TABLE public.transactions ADD COLUMN commission integer not null default 0;
ALTER TABLE public.transactions ADD COLUMN commission_converted integer not null default 0;
