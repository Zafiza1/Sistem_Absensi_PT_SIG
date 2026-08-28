import { SimpleNameCrud } from "@/components/crud/simple-name-crud";

export default function PositionsPage() {
  return (
    <SimpleNameCrud resource="positions" endpoint="/positions" singularLabel="Jabatan" pluralLabel="Jabatan" />
  );
}
