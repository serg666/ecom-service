alter table public.transactions alter column instrument set not null;
alter table public.transactions drop column sbpqr;
