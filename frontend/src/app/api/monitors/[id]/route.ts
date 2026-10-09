import { NextRequest } from "next/server";

import { handleMonitors } from "@/lib/auth-proxy";

type RouteContext = { params: Promise<{ id: string }> };

export async function PATCH(request: NextRequest, { params }: RouteContext) {
  const { id } = await params;
  return handleMonitors(request, `/api/monitors/${encodeURIComponent(id)}`);
}

export async function DELETE(request: NextRequest, { params }: RouteContext) {
  const { id } = await params;
  return handleMonitors(request, `/api/monitors/${encodeURIComponent(id)}`);
}
