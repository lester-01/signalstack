"use client";

import { useChat } from "@/providers/websocketProvider";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import Link from "next/link";
import { WebSocketStateEnum } from "@/providers/websocketProvider";
import { Button } from "@/components/ui/button";

export default function DashboardPage() {
  const { status: websocketStatus, authState, clientId, connectedClients } = useChat();

  return (
    <div className="flex h-screen flex-col gap-4 bg-slate-950 p-4">
      {/* Header */}
      <Card className="border-slate-800 bg-slate-900">
        <CardHeader>
          <div className="flex flex-col gap-4">
            <div className="flex items-center justify-between">
              <CardTitle className="text-2xl">Dashboard</CardTitle>
              <div className="text-sm text-slate-400">
                <span className="font-medium">Your ID:</span> <span className="text-slate-300">{clientId || "..."}</span>
              </div>
            </div>
            <div className="flex items-center gap-6">
              <div className="flex items-center gap-2">
                <span className="text-sm font-medium text-slate-400">Connection:</span>
                <Badge
                  variant={
                    websocketStatus === WebSocketStateEnum.OPEN
                      ? "default"
                      : websocketStatus === WebSocketStateEnum.CLOSED
                        ? "destructive"
                        : "secondary"
                  }
                >
                  {websocketStatus}
                </Badge>
              </div>
              <div className="flex items-center gap-2">
                <span className="text-sm font-medium text-slate-400">Auth:</span>
                <Badge
                  variant={
                    authState === "authenticated"
                      ? "default"
                      : authState === "unauthenticated"
                        ? "destructive"
                        : "secondary"
                  }
                >
                  {authState}
                </Badge>
              </div>
            </div>
            <div className="text-sm text-slate-400">
              <span className="font-medium">{connectedClients.length} users online</span>
            </div>
          </div>
        </CardHeader>
      </Card>

      {/* Info Card */}
      <Card className="border-slate-800 bg-slate-900">
        <CardHeader>
          <CardTitle>Testing WebSocket Persistence</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-slate-300">
            This page tests navigation between routes. Your WebSocket connection and authentication status should persist across page navigation.
          </p>
        </CardContent>
      </Card>

      {/* Navigation */}
      <div className="flex gap-2">
        <Button asChild variant="outline" className="border-slate-700 text-slate-300 hover:bg-slate-800">
          <Link href="/">
            Back to Chat
          </Link>
        </Button>
      </div>
    </div>
  );
}