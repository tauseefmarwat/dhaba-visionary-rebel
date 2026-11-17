<?php
namespace Tests\Feature;
use Tests\TestCase;
class StatsRouteTest extends TestCase { public function test_health_route_is_registered(): void { $this->getJson('/api/health')->assertOk()->assertJsonPath('status','ok'); } }
