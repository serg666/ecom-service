alter table public.transactions alter column instrument drop not null;
alter table public.transactions add column sbpqr jsonb;
