package net.tgburrin;

import java.io.IOException;

import java.util.ArrayList;

import java.util.concurrent.atomic.AtomicInteger;

import org.apache.commons.io.IOUtils;

import org.apache.http.HttpStatus;
import org.apache.http.HttpEntity;
import org.apache.http.HttpResponse;

import org.apache.http.client.methods.HttpGet;
import org.apache.http.client.methods.HttpPost;

import org.apache.http.impl.client.DefaultHttpClient;

import com.google.gson.stream.JsonReader;
import com.google.gson.JsonParser;
import com.google.gson.JsonElement;
import com.google.gson.JsonObject;
import com.google.gson.JsonArray;

public class Main extends Thread {
    private static AtomicInteger msgCounter = new AtomicInteger(0);
    private static int statsTimeSec = 5;

    private static String webHost = "webserver01:8080";
    private static String apiVersion = "v1";
    private static String urlBase = "/"+apiVersion+"/content";

    private static int numberOfClients = 48;
    private static int numberOfViews = 10000;

    private int threadNumber = 0;
    private String contentId = null;

    public Main () {
    }

    public Main ( int tn, String cid ) {
        threadNumber = tn;
        contentId = cid;
    }

    public void run () {
        System.out.println("Starting child "+threadNumber+" ("+contentId+")");
        DefaultHttpClient cli = new DefaultHttpClient();
        for ( int i = 0; i < numberOfViews; i++ ) {
            try {
                HttpPost req = new HttpPost("http://"+webHost+"/"+apiVersion+"/pageview/"+contentId);
                HttpResponse resp = cli.execute(req);

                if ( resp.getStatusLine().getStatusCode() != HttpStatus.SC_NO_CONTENT ) {
                    String body = IOUtils.toString(resp.getEntity().getContent(), "UTF-8");
                    System.err.println("Could not post pageview: "+body);
                    System.exit(1);
                }

                msgCounter.incrementAndGet();
            } catch (IOException e) {
                System.err.println("Could not make post request: "+e.getMessage());
                System.exit(1);
            }
        }
    }

    public static ArrayList<String> getContentIds () throws Exception {
        DefaultHttpClient cli = new DefaultHttpClient();
        HttpGet req = new HttpGet("http://"+webHost+"/"+urlBase);

        HttpResponse resp = cli.execute(req);
        String body = IOUtils.toString(resp.getEntity().getContent(), "UTF-8");

        if ( resp.getStatusLine().getStatusCode() != HttpStatus.SC_OK )
            throw new Exception("Could not get content list: "+body);

        ArrayList<String> contentIds = new ArrayList<String>();
        JsonObject payload = (new JsonParser().parse(body)).getAsJsonObject();
        if ( payload.has("data") && payload.get("data").isJsonArray()) {
            for ( JsonElement cid : payload.getAsJsonArray("data") )
                contentIds.add(cid.getAsJsonObject().get("id").getAsString());
        }

        return contentIds;
    }

    public static void main ( String[] args ) {
        ArrayList<String> contentIds = null;
        try {
            contentIds = getContentIds();
        } catch (Exception e) {
            System.err.println("Could not get content id list: "+e.getMessage());
            System.exit(1);
        }

        if ( contentIds != null ) {

            ArrayList<Main> children = new ArrayList<Main>();

            Thread thread = new Thread() {
                public void run() {
                    while ( true ) {
                        try {
                            Thread.sleep(statsTimeSec * 1000);
                        } catch (InterruptedException e) {
                            break;
                        }
                        int msgCount = msgCounter.getAndSet(0);
                        System.out.println(((float)msgCount / (float)statsTimeSec)+" msgs/sec");
                    }
                }
            };
            thread.start();

            int cidIdx = 0;
            for ( int i = 0; i < numberOfClients; i ++ ) {
                if ( cidIdx >= contentIds.size() )
                    cidIdx = 0;

                Main c = new Main(i+1, contentIds.get(cidIdx++));
                c.start();
                children.add(c);
            }

            for ( Main c : children ) {
                try {
                    c.join();
                } catch ( InterruptedException e ) {
                    System.err.println("Error from child: "+e.getMessage());
                }
            }

            thread.interrupt();
            try {
                thread.join();
            } catch (InterruptedException e) {}

        }

        System.exit(0);
    }
}
