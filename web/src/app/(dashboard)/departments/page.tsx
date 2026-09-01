import { SimpleNameCrud } from "@/components/crud/simple-name-crud";

export default function DepartmentsPage() {
  return (
    <SimpleNameCrud
      resource="departments"
      endpoint="/departments"
      singularLabel="Divisi"
      pluralLabel="Divisi"
    />
  );
}
