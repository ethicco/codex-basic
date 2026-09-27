import { NextRequest } from "next/server";

import { handleAvatar } from "@/lib/auth-proxy";

export async function GET(request: NextRequest) {
  return handleAvatar(request);
}

export async function POST(request: NextRequest) {
  return handleAvatar(request);
}
