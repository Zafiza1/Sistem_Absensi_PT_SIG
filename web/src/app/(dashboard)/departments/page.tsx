"use client";

import { Building2 } from "lucide-react";

import { SimpleNameCrud } from "@/components/crud/simple-name-crud";

export default function DepartmentsPage() {
  return (
    <SimpleNameCrud
      resource="departments"
      endpoint="/departments"
      singularLabel="Divisi"
      pluralLabel="Divisi"
      icon={Building2}
    />
  );
}
