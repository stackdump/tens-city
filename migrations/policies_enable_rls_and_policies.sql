-- Enable RLS on objects
ALTER TABLE public.objects ENABLE ROW LEVEL SECURITY;

-- Allow authenticated users to insert objects where owner_uuid == auth.uid()
CREATE POLICY insert_objects_auth ON public.objects
  FOR INSERT
  WITH CHECK (owner_uuid = auth.uid()::uuid);

-- Allow owners to select their objects
CREATE POLICY select_objects_owner ON public.objects
  FOR SELECT
  USING (owner_uuid = auth.uid()::uuid);

-- Allow owners to update their objects
CREATE POLICY update_objects_owner ON public.objects
  FOR UPDATE
  USING (owner_uuid = auth.uid()::uuid)
  WITH CHECK (owner_uuid = auth.uid()::uuid);

-- Allow owners to delete their objects
CREATE POLICY delete_objects_owner ON public.objects
  FOR DELETE
  USING (owner_uuid = auth.uid()::uuid);

-- Signatures: allow insert for authenticated (you may want to restrict to trusted backends)
ALTER TABLE public.signatures ENABLE ROW LEVEL SECURITY;
CREATE POLICY insert_signatures_auth ON public.signatures
  FOR INSERT
  WITH CHECK (EXISTS (SELECT 1 FROM public.objects o WHERE o.cid = signatures.cid AND o.owner_uuid = auth.uid()::uuid));
CREATE POLICY select_signatures_owner ON public.signatures
  FOR SELECT
  USING (EXISTS (SELECT 1 FROM public.objects o WHERE o.cid = public.signatures.cid AND o.owner_uuid = auth.uid()::uuid));