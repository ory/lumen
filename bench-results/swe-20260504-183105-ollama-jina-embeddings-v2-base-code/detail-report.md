# SWE-Bench Detail Report

Generated: 2026-05-04 23:04 UTC

---

## php-hard [php]

**Issue:** Jobs that fail inside a DB transaction are silently lost — no failed_jobs record, no JobFailed event

> We had a 'phantom' problem that jobs that failed, were not logged to the failed_jobs table.
> 
> After investigation, we found that when a job times out while executing inside a database transaction, the job is never recorded in the `failed_jobs` table and the `JobFailed` event is never dispatched.
> 
> The transaction rollback triggered by the timeout undoes the failed-job insert before it can be committed.
> 
> Reproduction:
> 
> ```php
> class MyJob implements ShouldQueue
> {
>     public $timeout = 5;
> 
>     public function handle(): void
>     {
>         DB::transaction(function (): void {
>             sleep(10);
>         });
>     }
> }
> ```
> 
> Dispatch this job. It will time out after 5 seconds, but:
> - No row appears in the `failed_jobs` table
> - The `JobFailed` event is never fired
> - The job simply vanishes without a trace
> 
> Expected: timeout should still result in a `failed_jobs` entry and a fired `JobFailed` event, even when the job was executing inside a transaction.
> 
> Actual: neither occurs; the job disappears silently.
> 
> Environment: Laravel 10.37.3, PHP 8.2.7, MariaDB 10.3.27

### Metrics

| Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost |
|----------|----------|-----------|------------|------------|------|
| **baseline** | 442.2s | 2181 | 987665 | 22365 | $1.0361 |
| **with-lumen** | 167.8s | 26 | 462349 | 9785 | $0.4175 |

### baseline

**Rating: Good**

```diff
diff --git a/src/Illuminate/Queue/Failed/DatabaseFailedJobProvider.php b/src/Illuminate/Queue/Failed/DatabaseFailedJobProvider.php
index 49cb3b9..491cf4c 100644
--- a/src/Illuminate/Queue/Failed/DatabaseFailedJobProvider.php
+++ b/src/Illuminate/Queue/Failed/DatabaseFailedJobProvider.php
@@ -59,7 +59,13 @@ public function log($connection, $queue, $payload, $exception)
 
         $exception = (string) mb_convert_encoding($exception, 'UTF-8');
 
-        return $this->getTable()->insertGetId(compact(
+        $db = $this->resolver->connection($this->database);
+
+        while ($db->transactionLevel() > 0) {
+            $db->rollBack();
+        }
+
+        return $db->table($this->table)->insertGetId(compact(
             'connection', 'queue', 'payload', 'exception', 'failed_at'
         ));
     }
diff --git a/src/Illuminate/Queue/Failed/DatabaseUuidFailedJobProvider.php b/src/Illuminate/Queue/Failed/DatabaseUuidFailedJobProvider.php
index b3192f2..45ebc39 100644
--- a/src/Illuminate/Queue/Failed/DatabaseUuidFailedJobProvider.php
+++ b/src/Illuminate/Queue/Failed/DatabaseUuidFailedJobProvider.php
@@ -55,7 +55,13 @@ public function __construct(ConnectionResolverInterface $resolver, $database, $t
      */
     public function log($connection, $queue, $payload, $exception)
     {
-        $this->getTable()->insert([
+        $db = $this->resolver->connection($this->database);
+
+        while ($db->transactionLevel() > 0) {
+            $db->rollBack();
+        }
+
+        $db->table($this->table)->insert([
             'uuid' => $uuid = json_decode($payload, true)['uuid'],
             'connection' => $connection,
             'queue' => $queue,
diff --git a/tests/Queue/DatabaseFailedJobProviderTest.php b/tests/Queue/DatabaseFailedJobProviderTest.php
index 31d1714..218b5e4 100644
--- a/tests/Queue/DatabaseFailedJobProviderTest.php
+++ b/tests/Queue/DatabaseFailedJobProviderTest.php
@@ -99,6 +99,18 @@ public function testCanFlushFailedJobs()
         $this->assertSame(0, $this->failedJobsTable()->count());
     }
 
+    public function testLogRollsBackOpenTransactionBeforeInserting()
+    {
+        $connection = $this->db->getConnection();
+
+        $connection->beginTransaction();
+
+        $this->provider->log('database', 'default', json_encode(['uuid' => (string) Str::uuid()]), new RuntimeException());
+
+        $this->assertSame(0, $connection->transactionLevel());
+        $this->assertSame(1, $this->failedJobsTable()->count());
+    }
+
     public function testCanProperlyLogFailedJob()
     {
         $uuid = Str::uuid();
diff --git a/tests/Queue/DatabaseUuidFailedJobProviderTest.php b/tests/Queue/DatabaseUuidFailedJobProviderTest.php
index c490037..c018f56 100644
--- a/tests/Queue/DatabaseUuidFailedJobProviderTest.php
+++ b/tests/Queue/DatabaseUuidFailedJobProviderTest.php
@@ -12,6 +12,33 @@
 
 class DatabaseUuidFailedJobProviderTest extends TestCase
 {
+    public function testLogRollsBackOpenTransactionBeforeInserting()
+    {
+        $db = new DB;
+        $db->addConnection([
+            'driver' => 'sqlite',
+            'database' => ':memory:',
+        ]);
+        $db->getConnection()->getSchemaBuilder()->create('failed_jobs', function (Blueprint $table) {
+            $table->uuid();
+            $table->text('connection');
+            $table->text('queue');
+            $table->longText('payload');
+            $table->longText('exception');
+            $table->timestamp('failed_at')->useCurrent();
+        });
+
+        $provider = new DatabaseUuidFailedJobProvider($db->getDatabaseManager(), 'default', 'failed_jobs');
+        $connection = $db->getConnection();
+
+        $connection->beginTransaction();
+
+        $provider->log('connection-1', 'queue-1', json_encode(['uuid' => 'uuid-1']), new RuntimeException());
+
+        $this->assertSame(0, $connection->transactionLevel());
+        $this->assertSame(1, $connection->table('failed_jobs')->count());
+    }
+
     public function testGettingIdsOfAllFailedJobs()
     {
         $provider = $this->getFailedJobProvider();
```

### with-lumen

**Rating: Good**

```diff
diff --git a/src/Illuminate/Queue/Jobs/Job.php b/src/Illuminate/Queue/Jobs/Job.php
index a41b401..72cd670 100755
--- a/src/Illuminate/Queue/Jobs/Job.php
+++ b/src/Illuminate/Queue/Jobs/Job.php
@@ -204,6 +204,19 @@ public function fail($e = null)
             }
         }
 
+        // If the exception is due to a job timing out, we need to rollback any open
+        // database transactions so that the failed_jobs record is not lost when the
+        // database rolls back the transaction after the process is killed.
+        if ($e instanceof TimeoutExceededException) {
+            try {
+                foreach ($this->resolve('db')->getConnections() as $connection) {
+                    $connection->rollBack(0);
+                }
+            } catch (Throwable) {
+                // ...
+            }
+        }
+
         try {
             // If the job has failed, we will delete it, call the "failed" method and then call
             // an event indicating the job has failed so it can be logged if needed. This is
```


