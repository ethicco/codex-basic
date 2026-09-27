import { NextRequest } from "next/server";

import { handleMonitors } from "@/lib/auth-proxy";

export async function GET(request: NextRequest) {
  return handleMonitors(request);
}

export async function POST(request: NextRequest) {
  return handleMonitors(request);
}
