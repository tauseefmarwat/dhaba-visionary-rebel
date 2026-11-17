package com.visionaryrebels.kitchen;

import java.time.Instant;
import java.util.Map;
import java.util.concurrent.TimeUnit;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.CommandLineRunner;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.stereotype.Component;

@SpringBootApplication
public class KitchenWorkerApplication {
    public static void main(String[] args) { SpringApplication.run(KitchenWorkerApplication.class, args); }
}

@Component
class QueueConsumer implements CommandLineRunner {
    private final StringRedisTemplate redis;
    private final JdbcTemplate jdbc;
    @Value("${worker.id}") private String workerId;
    @Value("${app.version}") private String version;
    QueueConsumer(StringRedisTemplate redis, JdbcTemplate jdbc) { this.redis=redis; this.jdbc=jdbc; }
    @Override public void run(String... args) {
        while (!Thread.currentThread().isInterrupted()) {
            try {
                redis.opsForValue().set("dhaba:worker:heartbeat", workerId + " version=" + version + " " + Instant.now());
                String raw = redis.opsForList().rightPopAndLeftPush("dhaba:orders:queue", "dhaba:orders:processing", 10, TimeUnit.SECONDS);
                if (raw == null) continue;
                long id = extractId(raw);
                if (id <= 0) { redis.opsForList().remove("dhaba:orders:processing", 1, raw); continue; }
                Integer changed = jdbc.update("UPDATE orders SET status='cooking',updated_at=NOW() WHERE id=? AND status='new'", id);
                if (changed > 0) jdbc.update("INSERT INTO order_status_history(order_id,old_status,new_status) VALUES(?, 'new','cooking')", id);
                Thread.sleep(2500);
                changed = jdbc.update("UPDATE orders SET status='ready',updated_at=NOW() WHERE id=? AND status='cooking'", id);
                if (changed > 0) jdbc.update("INSERT INTO order_status_history(order_id,old_status,new_status) VALUES(?, 'cooking','ready')", id);
                // Acknowledge only after PostgreSQL has recorded the terminal state.
                redis.opsForList().remove("dhaba:orders:processing", 1, raw);
            } catch (InterruptedException e) { Thread.currentThread().interrupt(); }
            catch (Exception e) { System.err.println("Worker processing error: " + e.getMessage()); try { Thread.sleep(1500); } catch (InterruptedException ex) { Thread.currentThread().interrupt(); } }
        }
    }
    private long extractId(String json) {
        try { String digits=json.replaceAll("[^0-9]", ""); return Long.parseLong(digits); }
        catch (Exception e) { return -1; }
    }
}
