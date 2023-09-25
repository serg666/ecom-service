BEGIN;
ALTER TABLE public.routes ADD COLUMN channel_id integer;
ALTER TABLE public.routes ADD CONSTRAINT routes_channel_id_fkey FOREIGN KEY (channel_id) REFERENCES public.channels (id) ON UPDATE RESTRICT ON DELETE RESTRICT;
ALTER TABLE public.transactions ADD COLUMN channel_id integer;
ALTER TABLE public.transactions ADD CONSTRAINT transactions_channel_id_fkey FOREIGN KEY (channel_id) REFERENCES public.channels (id) ON UPDATE RESTRICT ON DELETE RESTRICT;
UPDATE public.routes r SET channel_id = a.channel_id FROM public.routes r1 LEFT JOIN public.accounts a ON a.id=r1.account_id WHERE r.id=r1.id;
UPDATE public.transactions t SET channel_id = a.channel_id FROM public.transactions t1 LEFT JOIN public.accounts a ON a.id=t1.account_id WHERE t.id=t1.id;
ALTER TABLE public.transactions ALTER COLUMN channel_id SET not null;
ALTER TABLE public.routes ALTER COLUMN account_id SET not null;
ALTER TABLE public.routes ALTER COLUMN channel_id SET not null;

ALTER TABLE public.accounts DROP COLUMN channel_id;
COMMIT;
