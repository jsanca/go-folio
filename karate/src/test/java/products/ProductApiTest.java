package products;

import com.intuit.karate.junit5.Karate;

class ProductApiTest {

    @Karate.Test
    Karate testProducts() {
        return Karate.run("products").relativeTo(getClass());
    }
}
