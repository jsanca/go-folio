package catalog;

import com.intuit.karate.junit5.Karate;

class CatalogSyncTest {

    @Karate.Test
    Karate testCatalogSync() {
        return Karate.run("catalog_sync").relativeTo(getClass());
    }
}
